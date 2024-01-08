package metacritic

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/zestze/zest-backend/internal/zlog"

	"github.com/gin-gonic/gin"
)

func Register(r gin.IRouter) {
	g := r.Group("/metacritic")
	g.GET("/posts", getPostsForAPI)
	g.GET("/refresh", refresh)
}

func getPostsForAPI(c *gin.Context) {
	logger := zlog.Logger(c)

	opts := Options{}
	if err := c.BindQuery(&opts); err != nil {
		logger.Error("error binding query for getPosts", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide correct query params",
		})
		return
	} else if !opts.IsValid() {
		logger.Error("options not set correctly", "options", opts)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide correct query params",
		})
		return
	}

	logger = logger.With(opts.Group())

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

func refresh(c *gin.Context) {
	logger := zlog.Logger(c)

	currYear := time.Now().UTC().Year()
	const numPages = 5
	for _, m := range AvailableMediums {
		// just fetch for current year!
		for i := 1; i <= numPages; i++ {
			logger := logger.With(slog.String("medium", string(m)),
				slog.Int("year", currYear),
				slog.Int("page", i))

			logger.Info("fetching posts")
			posts, err := FetchPosts(c, Options{
				Medium:  m,
				MinYear: currYear,
				MaxYear: currYear,
				Page:    i,
			})
			if err != nil {
				logger.Error("error fetching posts", "error", err)
				c.IndentedJSON(http.StatusInternalServerError, gin.H{
					"error": "internal error",
				})
				return
			}

			logger.Info("persisting posts")
			ids, err := PersistPosts(c, posts)
			if err != nil {
				logger.Error("error persisting posts", "error", err)
				c.IndentedJSON(http.StatusInternalServerError, gin.H{
					"error": "internal error",
				})
				return
			}

			logger.Info("persisted " + strconv.Itoa(len(ids)) + " items")

			// ensure we don't get blacklisted
			logger.Info("going to sleep")
			time.Sleep(1 * time.Second)
		}
	}
}
