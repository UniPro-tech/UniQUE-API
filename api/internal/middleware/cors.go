package middleware

import (
	"time"

	configManager "github.com/UniPro-tech/UniQUE-API/api/internal/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	config, err := configManager.New()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}
	return cors.New(cors.Config{
		AllowOrigins: config.FrontendURLs,
		AllowMethods: []string{
			"POST",
			"GET",
			"DELETE",
			"OPTIONS",
			"PUT",
			"PATCH",
		},
		AllowHeaders: []string{
			"Content-Type",
		},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	})
}
