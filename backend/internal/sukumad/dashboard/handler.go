package dashboard

import (
	"context"
	"net/http"

	"basepro/backend/internal/apperror"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetOperations(c *gin.Context) {
	snapshot, err := h.service.GetOperationsSnapshot(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

func (h *Handler) StreamOperations(c *gin.Context) {
	websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		events, unsubscribe := h.service.SubscribeOperationsEvents(ctx)
		defer unsubscribe()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				if err := websocket.JSON.Send(conn, event); err != nil {
					return
				}
			}
		}
	}).ServeHTTP(c.Writer, c.Request)
}
