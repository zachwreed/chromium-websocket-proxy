package main

import (
	"chromium-websocket-proxy/chromepool"
	"chromium-websocket-proxy/chromeprofile"
	"chromium-websocket-proxy/config"
	"chromium-websocket-proxy/logger"
	"chromium-websocket-proxy/metrics"
	"chromium-websocket-proxy/proxyqueue"
	"chromium-websocket-proxy/servemux"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	c := config.Get()
	log := logger.Get()

	err := c.Validate()
	if err != nil {
		log.Fatal().Err(err).Msg("service configuration failed validation")
	}

	err = metrics.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to start metrics client")
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", c.GetServerConfig().Port))
	if err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("unable to listen on port %d", c.GetServerConfig().Port))

	}
	log.Info().Msg(fmt.Sprintf("listening on http://%v", l.Addr()))

	chromeprofile.LoadProfiles()

	crmPool := chromepool.Get()
	defer crmPool.ShutDownPool()

	s := &http.Server{
		Handler: servemux.NewServeMux(http.NewServeMux()),
		// TODO: determine what these should be set to, if anything
		ReadTimeout:  time.Second * 120,
		WriteTimeout: time.Second * 120,
	}

	errc := make(chan error, 1)
	go func() {
		errc <- s.Serve(l)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		log.Fatal().Err(err).Msg("failed to serve")
	case sig := <-sigs:
		log.Info().Msg(fmt.Sprintf("terminating with %v", sig))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	proxyqueue.Stop()

	err = s.Shutdown(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to gracefully terminate server")
	}
}
