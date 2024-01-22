package reddit

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zlog"
)

var (
	UserIdKey string = "zest.user_id"
)

type Controller struct {
	Client Client
	Store  Store
}

func New(db *sql.DB) (Controller, error) {
	client, err := NewClient(http.DefaultTransport)
	if err != nil {
		return Controller{}, err
	}
	return Controller{
		Client: client,
		Store:  NewStore(db),
	}, nil
}

func (svc Controller) Register(r gin.IRouter, auth gin.HandlerFunc) {
	g := r.Group("/reddit")
	g.Use(auth)
	g.GET("/posts", withParams(svc.getPosts))
	g.GET("/subreddits", withParams(svc.getSubreddits))
	g.POST("/refresh", withParams(svc.refresh))
}

func withParams(f func(*gin.Context, int, *slog.Logger)) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt(user.UserIdKey)
		logger := zlog.Logger(c)
		f(c, userID, logger)
	}
}

func (svc Controller) getPosts(c *gin.Context, userID int, logger *slog.Logger) {
	var (
		savedPosts []Post
		err        error
	)
	if subreddit := c.DefaultQuery("subreddit", "none"); subreddit != "none" {
		savedPosts, err = svc.Store.GetPostsFor(c, subreddit, userID)
	} else {
		savedPosts, err = svc.Store.GetAllPosts(c, userID)
	}

	if err != nil {
		logger.Error("error loading posts", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"posts": savedPosts,
	})
}

func (svc Controller) getSubreddits(c *gin.Context, userID int, logger *slog.Logger) {
	subreddits, err := svc.Store.GetSubreddits(c, userID)
	if err != nil {
		logger.Error("error loading subreddits", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"subreddits": subreddits,
	})
}

func (svc Controller) refresh(c *gin.Context, userID int, logger *slog.Logger) {
	savedPosts, err := svc.Client.Fetch(c, false)
	if err != nil {
		logger.Error("error fetching posts", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}

	logger.Info("successfully fetched posts", slog.Int("num_posts", len(savedPosts)))

	ids, err := svc.Store.PersistPosts(c, savedPosts, userID)
	if err != nil {
		logger.Error("error persisting posts", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}

	logger.Info("successfully persisted posts", slog.Int("num_persisted", len(ids)))

	c.IndentedJSON(http.StatusOK, gin.H{"num_refreshed": len(ids)})
}
