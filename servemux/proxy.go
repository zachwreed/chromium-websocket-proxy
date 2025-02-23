package servemux

import (
	"chromium-websocket-proxy/logger"
	"chromium-websocket-proxy/proxyqueue"
	"encoding/json"
	"fmt"
	"net/http"
)

func (sm *ServeMux) proxyHandler(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()
	log.Info().Ctx(r.Context()).Msg("queuing new chrome proxy session")

	pq := proxyqueue.Get()

	eld, err := proxyqueue.NewElementData(w, r)
	if err != nil {
		data := ServeResponse{
			id: -1,
			error: ServeResponseError{
				message: err.Error(),
				code:    -1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	el := pq.AddToList(eld)

	select {
	case status := <-eld.C:
		log.Info().Ctx(r.Context()).Msg(fmt.Sprintf("proxy finished with status: %s", status))
		return
	case <-eld.R.Context().Done():
		pq.RemoveFromList(el)
		return
	}
}
