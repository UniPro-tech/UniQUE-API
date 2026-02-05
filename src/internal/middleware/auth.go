package middleware

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/UniPro-tech/UniQUE-API/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWKS キャッシュ
var (
	jwksMu       sync.RWMutex
	jwksKeyCache = map[string]*rsa.PublicKey{}
	jwksFetched  time.Time
	httpClient   = &http.Client{Timeout: 5 * time.Second}
)

const jwksCacheTTL = 24 * time.Hour

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// swagger用のパスはスキップ
		if strings.HasPrefix(c.Request.URL.Path, "/swagger/") || c.Request.URL.Path == "/swagger.json" {
			c.Next()
			return
		}

		// AuthorizationヘッダーからJWTを取得
		authorization := c.GetHeader("Authorization")
		token := extractToken(authorization)
		if token == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		// トークン検証
		cfg := c.MustGet("config").(config.Config)
		claims := validateToken(token, c, cfg)
		if claims == nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token"})
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}

func extractToken(authorization string) string {
	// "Bearer <token>"形式からトークン部分を抽出
	const bearerPrefix = "Bearer "
	if len(authorization) > len(bearerPrefix) && authorization[:len(bearerPrefix)] == bearerPrefix {
		return authorization[len(bearerPrefix):]
	}
	return ""
}

// fetchJWKS fetches JWKS JSON and fills jwksKeyCache
func fetchJWKS(jwksURL string) error {
	resp, err := http.Get(jwksURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var body struct {
		Keys []map[string]interface{} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}
	tmp := map[string]*rsa.PublicKey{}
	for _, k := range body.Keys {
		if kty, _ := k["kty"].(string); kty != "RSA" {
			continue
		}
		kid, _ := k["kid"].(string)
		nStr, _ := k["n"].(string)
		eStr, _ := k["e"].(string)
		if kid == "" || nStr == "" || eStr == "" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
		if err != nil {
			continue
		}
		n := new(big.Int).SetBytes(nBytes)
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		tmp[kid] = &rsa.PublicKey{N: n, E: e}
	}
	jwksMu.Lock()
	jwksKeyCache = tmp
	jwksFetched = time.Now()
	jwksMu.Unlock()
	return nil
}

type AccessTokenClaims struct {
	jwt.RegisteredClaims
	Scope string `json:"scope,omitempty"`
}

func validateToken(token string, c *gin.Context, cfg config.Config) *jwt.RegisteredClaims {
	// issuer を config -> 環境変数 -> デフォルト の順で取得
	issuer := cfg.IssuerURL
	if issuer == "" {
		issuer = os.Getenv("CONFIG_ISSUER_URL")
	}
	if issuer == "" {
		issuer = "http://localhost:8080"
	}
	jwksURL := strings.TrimRight(issuer, "/") + "/.well-known/jwks.json"

	// keyfunc
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		kidRaw, ok := t.Header["kid"]
		if !ok {
			return nil, fmt.Errorf("token missing kid")
		}
		kid, ok := kidRaw.(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("invalid kid")
		}

		// キャッシュ確認
		jwksMu.RLock()
		pub := jwksKeyCache[kid]
		age := time.Since(jwksFetched)
		jwksMu.RUnlock()
		if pub != nil && age < jwksCacheTTL {
			return pub, nil
		}

		// 取得して再試行
		if err := fetchJWKS(jwksURL); err != nil {
			return nil, err
		}
		jwksMu.RLock()
		pub = jwksKeyCache[kid]
		jwksMu.RUnlock()
		if pub == nil {
			return nil, fmt.Errorf("public key not found for kid %s", kid)
		}
		return pub, nil
	}

	parsed, err := jwt.Parse(token, keyFunc, jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}), jwt.WithStrictDecoding())
	if err != nil {
		return nil
	}
	var claims *jwt.RegisteredClaims
	var ok bool
	sub, _ := parsed.Claims.GetSubject()
	if strings.HasPrefix(sub, "SID_") {
		claims, ok = parsed.Claims.(*jwt.RegisteredClaims)
	} else {
		c, ok2 := parsed.Claims.(*AccessTokenClaims)
		if ok2 {
			claims = &c.RegisteredClaims
			ok = true
		}
	}
	if !ok {
		return nil
	}

	// claimからの検証
	if claims.Issuer != issuer {
		return nil
	}
	if claims.ExpiresAt == nil || time.Until(claims.ExpiresAt.Time) <= 0 {
		return nil
	}
	if claims.NotBefore != nil && time.Until(claims.NotBefore.Time) < 0 {
		return nil
	}
	if claims.IssuedAt != nil && time.Until(claims.IssuedAt.Time.Add(time.Hour*24*7)) < 0 {
		return nil
	}
	var isValidToken bool = true
	if strings.HasPrefix(claims.Subject, "SID_") {
		isValidToken = verifyJTI(claims.ID, cfg, "/internal/session_verify")
	} else {
		isValidToken = verifyJTI(claims.ID, cfg, "/internal/token_verify")
	}
	if !isValidToken {
		return nil
	}
	return claims
}

type SessionVerifyResponse struct {
	Valid     bool      `json:"valid"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// verifyJTI calls the issuer's internal endpoint to verify a jti.
func verifyJTI(jti string, cfg config.Config, path string) bool {
	issuer := strings.TrimRight(cfg.IssuerURL, "/")
	url := issuer + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}
	q := req.URL.Query()
	q.Add("jti", jti)
	req.URL.RawQuery = q.Encode()
	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	var res struct {
		Valid     bool      `json:"valid"`
		ExpiresAt time.Time `json:"expires_at,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false
	}
	return res.Valid
}

type TokenVerifyResponse struct {
	Valid bool `json:"valid"`
}
