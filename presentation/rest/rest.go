package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/H3Cki/Plotrader/core/inbound"
)

var (
	exchangeNameHeader   = "X-Exchange-Name"
	exchangeConfigHeader = "X-Exchange-Config"
	exchangEnvVarHeader  = "X-Exchange-EnvVar"
)

func New(svc inbound.FollowService, addr string) http.Server {
	return http.Server{
		Addr:    addr,
		Handler: &handler{svc: svc},
	}
}

type handler struct {
	svc inbound.FollowService
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

		exchange, err := exchangeFromReq(r)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(err.Error()))
			return
		}

		req.Exchange = exchange

		ctx := context.Background()
		resp, err := h.svc.CreateFollow(ctx, req)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		respBytes, _ := json.Marshal(resp)
		rw.Write(respBytes)
	}
}

func exchangeFromReq(r *http.Request) (inbound.Exchange, error) {
	cfgStr := r.Header.Get(exchangeConfigHeader)
	cfgStr = strings.Trim(cfgStr, "\"")
	cfgStr = strings.ReplaceAll(cfgStr, "\\", "")

	configMap := map[string]any{}
	if err := json.Unmarshal([]byte(cfgStr), &configMap); err != nil {
		return inbound.Exchange{}, fmt.Errorf("error unmarshalling config: %v", err)
	}

	return inbound.Exchange{
		Name:      r.Header.Get(exchangeNameHeader),
		ConfigEnv: r.Header.Get(exchangEnvVarHeader),
		Config:    configMap,
	}, nil
}
