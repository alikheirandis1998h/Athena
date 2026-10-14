package server

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"base-backend/internal/auth"
	"base-backend/internal/handlers"
	"base-backend/internal/logger"
)

type Dependencies struct {
	TokenManager *auth.TokenManager
	AuthHandler  *handlers.AuthHandler
	UserHandler  *handlers.UserHandler
}

func NewRouter(env string, allowedOrigins []string, deps Dependencies) *gin.Engine {
	if env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(logger.Middleware())
	r.Use(corsMiddleware(allowedOrigins))

	r.GET("/health", handlers.HealthCheck)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", deps.AuthHandler.Register)
			authGroup.POST("/login", deps.AuthHandler.Login)
			authGroup.POST("/refresh", deps.AuthHandler.Refresh)
			authGroup.POST("/logout", deps.AuthHandler.Logout)
		}

		users := v1.Group("/users")
		users.Use(auth.RequireAuth(deps.TokenManager))
		{
			users.GET("/me", deps.UserHandler.Me)
		}
	}

	return r
}

// corsMiddleware allows the configured origins (e.g. the Expo web dev server or a deployed
// web build) to call this API from the browser - native clients aren't subject to CORS at all,
// this only matters for the web target.
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (slices.Contains(allowedOrigins, "*") || slices.Contains(allowedOrigins, origin)) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
