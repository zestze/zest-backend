package metacritic

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zgin"
	"github.com/zestze/zest-backend/internal/zlog"
	"golang.org/x/sync/errgroup"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	Client Client
	Store  Store
}

func New(db *sql.DB, rt http.RoundTripper) Controller {
	return Controller{
		Client: NewClient(rt),
		Store:  NewStore(db),
	}
}

func (svc Controller) Register(r gin.IRouter, auth gin.HandlerFunc) {
	g := r.Group("/metacritic")
	g.Use(auth)
	g.GET("/posts", svc.getPostsForAPI)
	g.POST("/refresh", svc.refresh)
	g.PATCH("/posts", zgin.WithUser(svc.savePosts))
}

type SavePostsInput struct {
	IDs    []int64 `json:"ids"`
	Action Action  `json:"action"`
}

func (svc Controller) savePosts(c *gin.Context, userID user.ID, logger *slog.Logger) {
	input := SavePostsInput{
		Action: SAVED, // TODO(zeke): for now, default to "saved"
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Error("error binding body", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide posts correctly",
		})
		return
	}

	if err := svc.Store.SavePostsForUser(
		c.Request.Context(), input.IDs, userID, input.Action); err != nil {
		logger.Error("error saving posts", "error", err)
		zgin.InternalError(c)
		return
	}

	c.Status(http.StatusCreated)
}

func (svc Controller) getPostsForAPI(c *gin.Context) {
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
	posts, err := svc.Store.GetPosts(c.Request.Context(), opts)
	if err != nil {
		slog.Error("error fetching posts", "error", err)
		zgin.InternalError(c)
		return
	}

	logger.Info("successfully fetched posts", slog.Int("num_posts", len(posts)))

	c.IndentedJSON(http.StatusOK, gin.H{
		"posts": posts,
	})
}

// since Store and Client both wrap *sq.DB and *http.Client
// can call worker thread on  Controller instead *Controller
func (svc Controller) worker(ctx context.Context, opts Options, logger *slog.Logger) error {
	logger = logger.With(opts.Group())
	logger.Info("fetching posts")
	posts, err := svc.Client.FetchPosts(ctx, opts)
	if err != nil {
		logger.Error("error fetching posts", "error", err)
		return err
	}

	logger.Info("persisting posts")
	ids, err := svc.Store.PersistPosts(ctx, posts)
	if err != nil {
		logger.Error("error persisting posts", "error", err)
		return err
	}

	logger.Info("persisted " + strconv.Itoa(len(ids)) + " items")

	// ensure we don't get blacklisted
	logger.Info("going to sleep")
	time.Sleep(1 * time.Second)
	return nil
}

func (svc Controller) refresh(c *gin.Context) {
	logger := zlog.Logger(c)

	currYear := time.Now().UTC().Year()
	const numPages = 5

	g, ctx := errgroup.WithContext(c.Request.Context())
	const HARDCODED_LIMIT = 20
	g.SetLimit(HARDCODED_LIMIT)

	for _, m := range AvailableMediums {
		for i := range numPages {
			if err := ctx.Err(); err != nil {
				logger.Error("error from workers", "error", err)
				zgin.InternalError(c)
				return
			}

			// shadow variables
			// not necessary in go1.22 but doing it just to be safe
			i, m := i, m
			g.Go(func() error {
				return svc.worker(ctx, Options{
					Page:    i + 1,
					Medium:  m,
					MinYear: currYear,
					MaxYear: currYear,
				}, logger)
			})
		}
	}

	if err := g.Wait(); err != nil {
		logger.Error("error from worker threads", "error", err)
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
