package spotify

import (
	"database/sql"
	"fmt"
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

	// TODO(zeke): persist!
	fmt.Printf("items are: %+v", items)

	c.IndentedJSON(http.StatusOK, gin.H{
		"num_refreshed": 0,
	})
}
