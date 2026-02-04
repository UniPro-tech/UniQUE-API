package main

import (
	"log"
	"net/http"

	"github.com/UniPro-tech/UniQUE-API/internal/config"
	"github.com/UniPro-tech/UniQUE-API/internal/db"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag/example/basic/docs"
	"gorm.io/gorm/logger"

	"github.com/gin-gonic/gin"
)

type HealthResponse struct {
	Status string `json:"status"`
}

// @BasePath /

// HealthCheck godoc
// @Summary health check endpoint
// @Schemes
// @Description システムの稼働状況を確認するためのエンドポイントです。
// @Tags system info
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

func main() {
	environmentConfigs := config.LoadConfig()

	// Initialize database
	dbConnection, err := db.NewDB()
	dbConnection.Logger = dbConnection.Logger.LogMode(logger.Info)
	if err != nil {
		log.Fatal(err)
	}

	// loggerとrecoveryミドルウェア付きGinルーター作成
	r := gin.Default()

	// Swagger Info
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Title = environmentConfigs.AppName + " Auth API"
	docs.SwaggerInfo.Version = environmentConfigs.Version

	// Add contexts
	r.Use(func(c *gin.Context) {
		c.Set("config", environmentConfigs)
		c.Set("db", dbConnection)
		c.Next()
	})

	// Routes
	r.GET("/health", healthCheck)

	// Start server
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	r.Run()
}
