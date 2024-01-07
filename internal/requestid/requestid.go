package requestid

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// based on https://github.com/gin-contrib/requestid
// slightly modified bc I want to use the standard context

var (
	headerXRequestID string
	contextRequestID string
)

// Config defines the config for RequestID middleware
type config struct {
	// Generator defines a function to generate an ID.
	// Optional. Default: func() string {
	//   return uuid.New().String()
	// }
	generator Generator
	headerKey HeaderStrKey
	handler   Handler
}

// New initializes the RequestID middleware.
func New(opts ...Option) gin.HandlerFunc {
	cfg := &config{
		generator: func() string {
			return uuid.New().String()
		},
		headerKey: "X-Request-ID",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	headerXRequestID = string(cfg.headerKey)

	return func(c *gin.Context) {
		// Get id from request
		rid := c.GetHeader(headerXRequestID)
		if rid == "" {
			rid = cfg.generator()
			c.Request.Header.Add(headerXRequestID, rid)
		}
		if cfg.handler != nil {
			cfg.handler(c, rid)
		}
		// crucial difference. Could also add as a handler but think
		// this is better.
		c.Set(contextRequestID, rid)
		// Set the id to ensure that the requestid is in the response
		c.Header(headerXRequestID, rid)
		c.Next()
	}
}

// Get returns the request identifier
func Get(c *gin.Context) string {
	return c.Writer.Header().Get(headerXRequestID)
}

// From grabs the request id from a context.Context
// only works if c.Set() is called ahead of time
func From(c context.Context) string {
	return c.Value(contextRequestID).(string)
}

// Option for queue system
type Option func(*config)

type (
	Generator func() string
	Handler   func(c *gin.Context, requestID string)
)

type HeaderStrKey string

// WithGenerator set generator function
func WithGenerator(g Generator) Option {
	return func(cfg *config) {
		cfg.generator = g
	}
}

// WithCustomeHeaderStrKey set custom header key for request id
func WithCustomHeaderStrKey(s HeaderStrKey) Option {
	return func(cfg *config) {
		cfg.headerKey = s
	}
}

// WithHandler set handler function for request id with context
func WithHandler(handler Handler) Option {
	return func(cfg *config) {
		cfg.handler = handler
	}
}
