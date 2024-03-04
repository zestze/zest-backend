package spotify

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zestze/zest-backend/internal/zgin"
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
	g := r.Group("/spotify")
	g.Use(auth)
	g.POST("/refresh", zgin.WithUser(svc.refresh))
	g.POST("/token", zgin.WithUser(svc.addToken))
	g.GET("/songs", zgin.WithUser(svc.getSongs))
	g.GET("/artists", zgin.WithUser(svc.getArtists))
	g.GET("/artist/songs", zgin.WithUser(svc.getSongsForArtist))
}

func (svc Controller) refresh(c *gin.Context, userID int, logger *slog.Logger) {
	token, err := svc.Store.GetToken(c, userID)
	if err != nil {
		logger.Error("error fetching token", "error", err)
		zgin.InternalError(c)
		return
	}

	logger.Info("successfully fetched token")
	if token.Expired() {
		token, err = svc.Client.RefreshAccess(c, token)
		if err != nil {
			logger.Error("error refreshing token", "error", err)
			zgin.InternalError(c)
			return
		}

		if err = svc.Store.PersistToken(c, token, userID); err != nil {
			logger.Error("error persisting token", "error", err)
			zgin.InternalError(c)
			return
		}
	}

	after := time.Now().Add(-time.Hour).UTC()
	items, err := svc.Client.GetRecentlyPlayed(c, token, after)
	if err != nil {
		logger.Error("error fetching songs", "error", err)
		zgin.InternalError(c)
		return
	}

	if len(items) == 0 {
		c.IndentedJSON(http.StatusOK, gin.H{
			"num_refreshed": 0,
		})
		return
	}

	persisted, err := svc.Store.PersistRecentlyPlayed(c, items, userID)
	if err != nil {
		logger.Error("error persisting songs", "error", err)
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"num_refreshed": len(persisted),
	})
}

func (svc Controller) addToken(c *gin.Context, userID int, logger *slog.Logger) {
	var token AccessToken
	if err := c.ShouldBindJSON(&token); err != nil {
		logger.Error("error binding body for token", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide token correctly",
		})
		return
	}

	if err := svc.Store.PersistToken(c, token, userID); err != nil {
		logger.Error("error persisting token", "error", err)
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusCreated, gin.H{
		"status": "ok",
	})
}

func (svc Controller) getSongs(c *gin.Context, userID int, logger *slog.Logger) {
	opts := defaultOptions()
	if err := c.BindQuery(&opts); err != nil {
		logger.Error("error binding query for getSongs", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide correct query params",
		})
		return
	}

	songs, err := svc.Store.GetRecentlyPlayed(c, userID, opts.Start, opts.End)
	if err != nil {
		logger.Error("error loading recently played songs", "error", err)
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"songs": songs,
	})
}

func (svc Controller) getArtists(c *gin.Context, userID int, logger *slog.Logger) {
	opts := defaultOptions()
	if err := c.BindQuery(&opts); err != nil {
		logger.Error("error binding query for getArtists", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide correct query params",
		})
		return
	}

	artists, err := svc.Store.GetRecentlyPlayedByArtist(c, userID, opts.Start, opts.End)
	if err != nil {
		logger.Error("error loading recently played artists", "error", err)
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"artists": artists,
	})
}

func (svc Controller) getSongsForArtist(c *gin.Context, userID int, logger *slog.Logger) {
	opts := struct {
		Options
		Artist string `form:"artist"`
	}{
		Options: defaultOptions(),
	}
	if err := c.BindQuery(&opts); err != nil || opts.Artist == "" {
		logger.Error("error binding query for getSongsByArtist", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide correct query params",
		})
		return
	}

	songs, err := svc.Store.GetRecentlyPlayedForArtist(c, userID, opts.Artist, opts.Start, opts.End)
	if err != nil {
		logger.Error("error loading recently played songs for artist", "error", err)
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"songs": songs,
	})
}

type Options struct {
	Start time.Time `form:"start"`
	End   time.Time `form:"end"`
}

func defaultOptions() Options {
	now := time.Now().UTC()
	return Options{
		Start: now.Add(-time.Hour),
		End:   now,
	}
}
