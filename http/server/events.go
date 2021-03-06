package server

import (
	"net/http"
	"strconv"

	"github.com/keys-pub/keys-ext/http/api"
	"github.com/keys-pub/keys/dstore/events"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func (s *Server) events(c echo.Context, path string, max int) (*api.EventsResponse, error) {
	request := c.Request()
	ctx := request.Context()

	var index int
	if f := c.QueryParam("idx"); f != "" {
		i, err := strconv.Atoi(f)
		if err != nil {
			return nil, ErrResponse(c, http.StatusBadRequest, errors.Wrapf(err, "invalid index"))
		}
		index = i
	}
	var limit int
	if f := c.QueryParam("limit"); f != "" {
		n, err := strconv.Atoi(f)
		if err != nil {
			return nil, ErrResponse(c, http.StatusBadRequest, errors.Wrapf(err, "invalid limit"))
		}
		limit = n
	}

	if limit == 0 || limit > max {
		limit = max
	}

	pdir := c.QueryParam("dir")
	if pdir == "" {
		pdir = "asc"
	}

	var dir events.Direction
	switch pdir {
	case "asc":
		dir = events.Ascending
	case "desc":
		dir = events.Descending
	default:
		return nil, ErrResponse(c, http.StatusBadRequest, errors.Errorf("invalid dir"))
	}

	s.logger.Infof("Events %s (from=%d)", path, index)
	iter, err := s.fi.Events(ctx, path, events.Index(int64(index)), events.Limit(int64(limit)), events.WithDirection(dir))
	if err != nil {
		return nil, s.internalError(c, err)
	}
	defer iter.Release()
	to := int64(index)
	events := []*events.Event{}
	for {
		event, err := iter.Next()
		if err != nil {
			return nil, s.internalError(c, err)
		}
		if event == nil {
			break
		}
		events = append(events, event)
		to = event.Index
	}
	s.logger.Infof("Events %s, got %d, (to=%d)", path, len(events), to)

	return &api.EventsResponse{
		Events: events,
		Index:  to,
	}, nil
}
