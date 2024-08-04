package zgin

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zlog"
)

type handler func(*gin.Context, user.ID, *slog.Logger)

func WithUser(f handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetInt(user.UserIdKey)
		logger := zlog.Logger(c)
		f(c, user.ID(id), logger)
	}
}

func InternalError(c *gin.Context) {
	c.IndentedJSON(http.StatusInternalServerError, gin.H{
		"error": "internal error",
	})
}

func BadRequest(c *gin.Context, message string) {
	c.IndentedJSON(http.StatusBadRequest, gin.H{
		"error": message,
	})
}
