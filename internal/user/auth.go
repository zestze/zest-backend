package user

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	maxAge time.Duration = 5 * time.Minute
)

type Session map[string]struct {
	User      User
	ExpiresAt time.Time
}

func NewSession() Session {
	return make(Session)
}

func (sess Session) IsActive(sessionID string) bool {
	item, ok := sess[sessionID]
	if ok && time.Now().After(item.ExpiresAt) {
		// remove!
		ok = false
		delete(sess, sessionID)
	}
	return ok
}

func (sess Session) Start(user User, expiresAt time.Time) string {
	// generate an ID!
	id := uuid.New().String()
	sess[id] = struct {
		User      User
		ExpiresAt time.Time
	}{
		User:      user,
		ExpiresAt: expiresAt,
	}
	return id
}

func (sess Session) GetUser(sessionID string) (User, bool) {
	item, ok := sess[sessionID]
	if !ok {
		return User{}, false
	}
	return item.User, true
}

func Auth(session Session) gin.HandlerFunc {
	return func(c *gin.Context) {

		token, err := c.Cookie(CookieName)
		if err != nil && errors.Is(err, http.ErrNoCookie) { // only possible err
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "please login to access",
			})
			return
		}

		if !session.IsActive(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		c.Next()
	}
}
