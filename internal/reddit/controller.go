package reddit

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zestze/zest-backend/internal/zlog"
)

type Controller struct {
	io.Closer
	Client Client
	Store  Store
}

func New() (Controller, error) {
	store, err := NewStore(DB_FILE_NAME)
	if err != nil {
		return Controller{}, err
	}
	client, err := NewClient(http.DefaultTransport)
	if err != nil {
		return Controller{}, err
	}
	return Controller{
		Client: client,
		Store:  store,
	}, nil
}

func (svc Controller) Close() error {
	return svc.Store.Close()
}

func (svc Controller) Register(r gin.IRouter) {
	g := r.Group("/reddit")
	g.GET("/posts", svc.getPosts)
	g.GET("/subreddits", svc.getSubreddits)
	g.POST("/refresh", svc.refresh)
}
func (svc Controller) getPosts(c *gin.Context) {
	logger := zlog.Logger(c)
	var (
		savedPosts []Post
		err        error
	)
	if subreddit := c.DefaultQuery("subreddit", "none"); subreddit != "none" {
		savedPosts, err = svc.Store.GetPostsFor(c, subreddit)
	} else {
		savedPosts, err = svc.Store.GetAllPosts(c)
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

func (svc Controller) getSubreddits(c *gin.Context) {
	logger := zlog.Logger(c)
	subreddits, err := svc.Store.GetSubreddits(c)
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

func (svc Controller) refresh(c *gin.Context) {
	logger := zlog.Logger(c)
	savedPosts, err := svc.Client.Fetch(c, false)
	if err != nil {
		logger.Error("error fetching posts", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}

	logger.Info("successfully fetched posts", slog.Int("num_posts", len(savedPosts)))

	ids, err := svc.Store.PersistPosts(c, savedPosts)
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
