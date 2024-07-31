package spotify

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/zestze/zest-backend/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/zestze/zest-backend/internal/zgin"
)

type Controller struct {
	Client    Client
	StoreV1   GeneralStore
	StoreV2   GeneralStore
	Publisher Publisher
}

func New(ctx context.Context, db *sql.DB, publisher Publisher) (Controller, error) {
	client, err := NewClient(http.DefaultTransport)
	if err != nil {
		return Controller{}, err
	}
	return Controller{
		Client:    client,
		StoreV1:   NewStoreV1(db),
		StoreV2:   NewStoreV2(db),
		Publisher: publisher,
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

func (svc Controller) refresh(c *gin.Context, userID user.ID, logger *slog.Logger) {
	token, err := svc.StoreV2.GetToken(c, userID)
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

		if err = svc.StoreV2.PersistToken(c, token, userID); err != nil {
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

	msg := gin.H{
		"num_persisted": 0,
	}
	if len(items) == 0 {
		if err = svc.Publisher.Publish(c, msg); err != nil {
			logger.Error("error publishing message", "error", err)
		}
		c.IndentedJSON(http.StatusOK, msg)
		return
	}

	// persist songs via both methods!
	persisted, err := svc.StoreV1.PersistRecentlyPlayed(c, items, userID)
	if err != nil {
		logger.Error("error persisting songs", "error", err)
		zgin.InternalError(c)
		return
	}

	_, err = svc.StoreV2.PersistRecentlyPlayed(c, items, userID)
	if err != nil {
		logger.Error("error persisting songs", "error", err)
		zgin.InternalError(c)
		return
	}

	msg["num_persisted"] = len(persisted)
	if err = svc.Publisher.Publish(c, msg); err != nil {
		logger.Error("error publishing message", "error", err)
	}
	c.IndentedJSON(http.StatusOK, msg)
}

func (svc Controller) addToken(c *gin.Context, userID user.ID, logger *slog.Logger) {
	var token AccessToken
	if err := c.ShouldBindJSON(&token); err != nil {
		logger.Error("error binding body for token", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide token correctly",
		})
		return
	}

	if err := svc.StoreV2.PersistToken(c, token, userID); err != nil {
		logger.Error("error persisting token", "error", err)
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusCreated, gin.H{
		"status": "ok",
	})
}

func (svc Controller) getSongs(c *gin.Context, userID user.ID, logger *slog.Logger) {
	opts := defaultOptions()
	if err := c.BindQuery(&opts); err != nil {
		logger.Error("error binding query for getSongs", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide correct query params",
		})
		return
	}

	songs, err := svc.StoreV2.GetRecentlyPlayed(c, userID, opts.Start, opts.End)
	if err != nil {
		logger.Error("error loading recently played songs", "error", err)
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"songs": songs,
	})
}

func (svc Controller) getArtists(c *gin.Context, userID user.ID, logger *slog.Logger) {
	opts := defaultOptions()
	if err := c.BindQuery(&opts); err != nil {
		logger.Error("error binding query for getArtists", "error", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "please provide correct query params",
		})
		return
	}

	artists, err := svc.StoreV2.GetRecentlyPlayedByArtist(c, userID, opts.Start, opts.End)
	if err != nil {
		logger.Error("error loading recently played artists", "error", err)
		zgin.InternalError(c)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"artists": artists,
	})
}

func (svc Controller) getSongsForArtist(c *gin.Context, userID user.ID, logger *slog.Logger) {
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

	songs, err := svc.StoreV2.GetRecentlyPlayedForArtist(c, userID, opts.Artist, opts.Start, opts.End)
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

// GeneralStore is a minimal abstraction over the two Store structs.
// this is so that it's clear to the controller what methods are available
type GeneralStore interface {
	PersistRecentlyPlayed(
		ctx context.Context, songs []PlayHistoryObject, userID int,
	) ([]string, error)
	GetRecentlyPlayed(
		ctx context.Context, userID int, start, end time.Time,
	) ([]NameWithTime, error)
	GetRecentlyPlayedByArtist(
		ctx context.Context, userID int, start, end time.Time,
	) ([]NameWithListens, error)
	GetRecentlyPlayedForArtist(
		ctx context.Context, userID int, artist string, start, end time.Time,
	) ([]NameWithListens, error)
	PersistToken(ctx context.Context, token AccessToken, userID int) error
	GetToken(ctx context.Context, userID int) (AccessToken, error)
}

type Publisher interface {
	Publish(ctx context.Context, message any) error
}
