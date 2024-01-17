package user

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"github.com/zestze/zest-backend/internal/zlog"
)

var (
	ErrInvalidIP error = errors.New("invalid IP")
)

type item struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	ClientIP string `json:"client_ip"`
}

type Session struct {
	*redis.Client
	MaxAge time.Duration
}

// connecting locally so should be fine to set no password
func NewSession(maxAge time.Duration) Session {
	return Session{
		Client: redis.NewClient(&redis.Options{
			Addr:     "redis:6379",
			Password: "",
			DB:       0,
		}),
		MaxAge: maxAge,
	}
}

func (sess Session) IsActive(
	ctx context.Context, sessionID, clientIP string,
) (bool, error) {
	value, err := sess.Get(ctx, sessionID).Result()
	if err != nil && errors.Is(err, redis.Nil) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	var item item
	if err := jsoniter.UnmarshalFromString(value, &item); err != nil {
		return false, err
	} else if item.ClientIP != clientIP {
		return false, ErrInvalidIP
	}

	return true, nil
}

func (sess Session) Start(
	ctx context.Context, user User, clientIP string,
) (string, error) {
	id := uuid.New().String()

	value, err := jsoniter.MarshalToString(item{
		UserID:   user.ID,
		Username: user.Username,
		ClientIP: clientIP,
	})
	if err != nil {
		return "", err
	}

	return id, sess.Set(ctx, id, value, sess.MaxAge).Err()
}

func (sess Session) GetUser(
	ctx context.Context, sessionID, clientIP string,
) (User, error) {
	value, err := sess.Get(ctx, sessionID).Result()
	if err != nil {
		return User{}, err
	}

	var item item
	if err := jsoniter.UnmarshalFromString(value, &item); err != nil {
		return User{}, err
	} else if item.ClientIP != clientIP {
		return User{}, ErrInvalidIP
	}

	return User{
		ID:       item.UserID,
		Username: item.Username,
	}, nil
}

func Auth(sess Session) gin.HandlerFunc {
	return func(c *gin.Context) {

		token, err := c.Cookie(CookieName)
		if err != nil && errors.Is(err, http.ErrNoCookie) { // only possible err
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "please login to access",
			})
			return
		}

		ok, err := sess.IsActive(c, token, c.ClientIP())
		if err != nil && !errors.Is(err, ErrInvalidIP) {
			zlog.Logger(c).Error("error when validating session", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "internal error",
			})
			return
		} else if err != nil || !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		c.Next()
	}
}
