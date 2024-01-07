package reddit

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zestze/zest-backend/internal/zlog"
)

func Register(r gin.IRouter) {
	g := r.Group("/reddit")
	g.GET("/posts", getPosts)
	g.GET("/subreddits", getSubreddits)
	g.POST("/refresh", refresh)
}
func getPosts(c *gin.Context) {
	logger := zlog.Logger(c)
	var (
		savedPosts []Post
		err        error
	)
	if subreddit := c.DefaultQuery("subreddit", "none"); subreddit != "none" {
		savedPosts, err = GetPostsFor(c, subreddit)
	} else {
		savedPosts, err = GetAllPosts(c)
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

func getSubreddits(c *gin.Context) {
	logger := zlog.Logger(c)
	subreddits, err := GetSubreddits(c)
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

func refresh(c *gin.Context) {
	logger := zlog.Logger(c)
	savedPosts, err := Fetch(c, false)
	if err != nil {
		logger.Error("error fetching posts", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}

	logger.Info("successfully fetched posts", slog.Int("num_posts", len(savedPosts)))

	ids, err := PersistPosts(c, savedPosts)
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
