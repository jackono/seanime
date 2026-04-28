package handlers

import (
	"errors"
	"seanime/internal/platforms/malcollection"

	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

// HandleSetTrackerMode
//
//	@summary sets the active tracking service.
//	@desc The selected tracker is persisted to config.toml and used for collection/progress operations.
//	@route /api/v1/tracker/mode [POST]
//	@returns handlers.Status
func (h *Handler) HandleSetTrackerMode(c echo.Context) error {
	type body struct {
		Mode string `json:"mode"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	mode := malcollection.NormalizeTrackerMode(b.Mode)
	if mode == malcollection.TrackerMAL {
		if _, err := h.App.Database.GetMalInfo(); err != nil {
			return h.RespondWithError(c, errors.New("connect MyAnimeList before selecting it as the tracker"))
		}
	} else {
		if _, err := h.App.Database.GetAccount(); err != nil {
			return h.RespondWithError(c, errors.New("connect AniList before selecting it as the tracker"))
		}
	}

	h.App.Config.Server.TrackerMode = mode
	malcollection.SetTrackerMode(mode)
	viper.Set("server.trackerMode", mode)
	if err := viper.WriteConfig(); err != nil {
		return h.RespondWithError(c, err)
	}

	go h.App.RefreshAnimeCollection()

	return h.RespondWithData(c, h.NewStatus(c))
}
