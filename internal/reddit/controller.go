package reddit

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

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
	g.POST("/backfill", zgin.WithUser(svc.backfill))
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

func (svc Controller) backfill(c *gin.Context, userID int, logger *slog.Logger) {
	// TODO(zeke): to have this, the client needs to be updated.
	// generally need to pass a start/stop.
	// I _think_ the way it works, is
	// saved posts are turned with a SORT BY created LIMIT 50
	// of sorts. And then providing an `after` param will give you the next with a
	// WHERE created_at < after.datetime
	// or something.
	//
	// for now, just do a refresh with all! will likely take a while, but hopefully isn't so bad.
	go func() {
		// TODO(zeke): create a new context?
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
		defer cancel()
		savedPosts, err := svc.Client.Fetch(ctx, true)
		if err != nil {
			logger.Error("error fetching posts", "error", err)
			return
		}

		logger.Info("successfully fetched posts", slog.Int("num_posts", len(savedPosts)))

		ids, err := svc.Store.PersistPosts(ctx, savedPosts, userID)
		if err != nil {
			logger.Error("error persisting posts", "error", err)
			return
		}

		logger.Info("successfully persisted posts", slog.Int("num_persisted", len(ids)))
	}()

	c.IndentedJSON(http.StatusAccepted, gin.H{
		"message": "backfill successfully started",
	})
}
