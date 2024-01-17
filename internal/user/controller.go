package user

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/zestze/zest-backend/internal/zlog"
	"golang.org/x/crypto/bcrypt"
)

var SALT = 8
var CookieName = "zest-token"

type Controller struct {
	io.Closer
	Store Store
	sess  Session
}

func New(sess Session) (Controller, error) {
	store, err := NewStore(DB_FILE_NAME)
	if err != nil {
		return Controller{}, err
	}
	return Controller{
		Store: store,
		sess:  sess,
	}, nil
}

func (svc Controller) Register(r gin.IRouter) {
	r.POST("/login", svc.Login)
	r.POST("/signup", svc.Signup)
	r.POST("/refresh", svc.Refresh)
}

func (svc Controller) Login(c *gin.Context) {
	logger := zlog.Logger(c)

	var (
		creds Credentials
		err   error
	)
	if err = c.ShouldBindJSON(&creds); err != nil {
		logger.Error("error binding body for login", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide credentials correctly",
		})
		return
	}

	// compare username password in store!
	user, err := svc.Store.GetUser(c, creds.Username)
	if err != nil {
		logger.Error("error fetching password for login", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "db error",
		})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password))
	if err != nil && errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{
			"status": "unauthorized",
		})
		return
	} else if err != nil {
		logger.Error("error when validating password", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error when validating password",
		})
		return
	}

	token, err := svc.sess.Start(c, user, c.ClientIP())
	if err != nil {
		logger.Error("error when starting session for user", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error when starting user session",
		})
		return
	}

	svc.setCookie(c, token, time.Now().Add(svc.sess.MaxAge))
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (svc Controller) Signup(c *gin.Context) {
	logger := zlog.Logger(c)

	var (
		creds Credentials
		err   error
	)
	if err = c.ShouldBindJSON(&creds); err != nil {
		// TODO(zeke): actually, should i be logging on these?
		logger.Error("error binding body for signup", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide credentials correctly",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), SALT)
	if err != nil {
		logger.Error("error generating hash for new user", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}

	id, err := svc.Store.PersistUser(c, creds.Username, string(hash), SALT)
	if err != nil {
		logger.Error("error persisting user", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}
	c.IndentedJSON(http.StatusCreated, gin.H{
		"user_id": id,
	})
}

func (svc Controller) Refresh(c *gin.Context) {
	logger := zlog.Logger(c)
	token, err := c.Cookie(CookieName)
	if err != nil && errors.Is(err, http.ErrNoCookie) { // only possible err
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "please login to access",
		})
		return
	}

	clientIP := c.ClientIP()
	user, err := svc.sess.GetUser(c, token, clientIP)
	if err != nil && (errors.Is(err, redis.Nil) || errors.Is(err, ErrInvalidIP)) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token",
		})
		return
	} else if err != nil {
		logger.Error("error grabbing user session", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}

	newToken, err := svc.sess.Start(c, user, clientIP)
	if err != nil {
		logger.Error("error starting user session", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}

	svc.setCookie(c, newToken, time.Now().Add(svc.sess.MaxAge))
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// can use c.SetCookie but it's just an annoying wrapper for this direct call
// might add more fields later
func (svc Controller) setCookie(c *gin.Context, value string, expiresAt time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:    CookieName,
		Value:   value,
		Expires: expiresAt,
	})
}

type Credentials struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (svc Controller) Close() error {
	return nil
}
