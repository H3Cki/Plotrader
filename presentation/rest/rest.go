package rest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/H3Cki/Plotrader/core/inbound"
)

func New(svc inbound.FollowerService, addr string) http.Server {
	return http.Server{
		Addr:    addr,
		Handler: &handler{svc: svc},
	}
}

type handler struct {
	svc inbound.FollowerService
}

func (h *handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		req := inbound.CreateFollowRequest{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		ctx := context.Background()

		if err := h.svc.StartFollow(ctx, req); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(err.Error()))
		}
	}
}
