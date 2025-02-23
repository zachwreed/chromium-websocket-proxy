package servemux

import (
	"chromium-websocket-proxy/config"
	"chromium-websocket-proxy/logger"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
)

type IHttpMux interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	Handler(r *http.Request) (h http.Handler, pattern string)
	Handle(pattern string, handler http.Handler)
}

type ServeRequest = func(http.ResponseWriter, *http.Request)

type ServeResponse struct {
	id    int
	error ServeResponseError
}

type ServeResponseError struct {
	code    int
	message string
}

type ServeMux struct {
	mux IHttpMux
}

func NewServeMux(
	mux IHttpMux,
) *ServeMux {
	sm := &ServeMux{
		mux: mux,
	}
	sm.mux.HandleFunc("/healthcheck", sm.healthCheck)
	sm.mux.HandleFunc("/connect", sm.accessTokenMiddleware(sm.proxyHandler))
	return sm
}

func (sm *ServeMux) accessTokenMiddleware(f ServeRequest) ServeRequest {
	return func(w http.ResponseWriter, r *http.Request) {
		if !config.Get().GetServerConfig().AccessTokenValidationEnabled {
			f(w, r)
			return
		}

		accessToken := r.URL.Query().Get("accessToken")

		if config.Get().GetServerConfig().AccessToken != accessToken {
			data := ServeResponse{
				id: -1,
				error: ServeResponseError{
					message: fmt.Sprintf("req.query['accessToken'] does not match required %s token", config.ServerAccessToken),
					code:    -1,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(data)
			return
		}
		f(w, r)
	}
}

func (sm *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionId := uuid.New()
	ctx := context.WithValue(r.Context(), logger.SessionIdTrackingKey, sessionId)
	rc := r.WithContext(ctx)
	sm.mux.ServeHTTP(w, rc)
}

func (sm *ServeMux) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
