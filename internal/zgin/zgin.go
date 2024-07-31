package zgin

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zlog"
)

func WithUser(f func(*gin.Context, int, *slog.Logger)) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt(user.UserIdKey)
		logger := zlog.Logger(c)
		f(c, userID, logger)
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
