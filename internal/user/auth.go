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
	redistrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/redis/go-redis.v9"
)

var (
	ErrInvalidIP = errors.New("invalid IP")
	UserIdKey    = "zest.user_id"
)

// TODO(zeke): mostly for external use right now
// TODO(zeke): should this be type alias vs type def?
type ID = int

type item struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	ClientIP string `json:"client_ip"`
}

type Session struct {
	redis.UniversalClient
	MaxAge time.Duration
}

func NewSession(opts ...RedisOption) Session {
	cfg := defaultRedisConfig()
	for _, o := range opts {
		o(&cfg)
	}

	var client redis.UniversalClient = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if cfg.tracing {
		redistrace.WrapClient(client)
	}
	return Session{
		UniversalClient: client,
		MaxAge:          cfg.MaxAge,
	}
}

type redisConfig struct {
	Addr     string
	Password string
	DB       int
	MaxAge   time.Duration
	tracing  bool
}

func defaultRedisConfig() redisConfig {
	return redisConfig{
		Addr: "redis:6379",
		// connecting locally so should be fine to set no password
		Password: "",
		DB:       0,
	}
}

type RedisOption func(cfg *redisConfig)

func WithTracing() RedisOption {
	return func(cfg *redisConfig) {
		cfg.tracing = true
	}
}

func WithMaxAge(maxAge time.Duration) RedisOption {
	return func(cfg *redisConfig) {
		cfg.MaxAge = maxAge
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

		user, err := sess.GetUser(c.Request.Context(), token, c.ClientIP())
		if err != nil && (errors.Is(err, ErrInvalidIP) || errors.Is(err, redis.Nil)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		} else if err != nil {
			zlog.Logger(c).Error("error when validating session", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "internal error",
			})
			return
		}

		// insert user id into context
		c.Set(UserIdKey, user.ID)

		c.Next()
	}
}
