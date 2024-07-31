package reddit

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zgin"
)

type Controller struct {
	Client api
	Store  Store
}

func New(db *sql.DB) (Controller, error) {
	secrets, err := loadSecrets(defaultSecretsPath)
	if err != nil {
		return Controller{}, err
	}
	return Controller{
		Client: NewClient(WithSecrets(secrets)),
		Store:  NewStore(db),
	}, nil
}

type api interface {
	Fetch(ctx context.Context, grabAll bool) ([]Post, error)
}

func (svc Controller) Register(r gin.IRouter, auth gin.HandlerFunc) {
	g := r.Group("/reddit")
	g.Use(auth)
	g.GET("/posts", zgin.WithUser(svc.getPosts))
	g.GET("/subreddits", zgin.WithUser(svc.getSubreddits))
	g.POST("/refresh", zgin.WithUser(svc.refresh))
}

func (svc Controller) getPosts(c *gin.Context, userID user.ID, logger *slog.Logger) {
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
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"posts": savedPosts,
	})
}

func (svc Controller) getSubreddits(c *gin.Context, userID user.ID, logger *slog.Logger) {
	subreddits, err := svc.Store.GetSubreddits(c, userID)
	if err != nil {
		logger.Error("error loading subreddits", "error", err)
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"subreddits": subreddits,
	})
}

func (svc Controller) refresh(c *gin.Context, userID user.ID, logger *slog.Logger) {
	savedPosts, err := svc.Client.Fetch(c, false)
	if err != nil {
		logger.Error("error fetching posts", "error", err)
		zgin.InternalError(c)
		return
	}

	logger.Info("successfully fetched posts", slog.Int("num_posts", len(savedPosts)))

	ids, err := svc.Store.PersistPosts(c, savedPosts, userID)
	if err != nil {
		logger.Error("error persisting posts", "error", err)
		zgin.InternalError(c)
		return
	}

	logger.Info("successfully persisted posts", slog.Int("num_persisted", len(ids)))

	c.IndentedJSON(http.StatusOK, gin.H{"num_refreshed": len(ids)})
}
