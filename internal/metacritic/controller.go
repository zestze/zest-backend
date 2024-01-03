package metacritic

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Register(r gin.IRouter) {
	g := r.Group("/metacritic")
	g.GET("/posts", getPostsForAPI)
}

func getPostsForAPI(c *gin.Context) {
	opts := Options{}
	if err := c.BindQuery(&opts); err != nil {
		slog.Error("error binding query for getPosts", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide correct query params",
		})
		return
	} else if !opts.IsValid() {
		slog.Error("options not set correctly", "options", opts)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide correct query params",
		})
		return
	}

	logger := slog.With(opts.Group())

	logger.Info("going to fetch posts")
	posts, err := GetPosts(c, opts)
	if err != nil {
		slog.Error("error fetching posts", "error", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}

	logger.Info("successfully fetched posts", slog.Int("num_posts", len(posts)))

	c.IndentedJSON(http.StatusOK, gin.H{
		"posts": posts,
	})
}
