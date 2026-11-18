package httpapi

import (
	"github.com/gin-gonic/gin"
)

func NewRouter(s *Server) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", s.Health)
	r.GET("/ws", s.WS)

	internal := r.Group("/internal/v1")
	internal.Use(s.requireInternalKey)
	{
		internal.POST("/notify", s.Notify)
	}

	return r
}
