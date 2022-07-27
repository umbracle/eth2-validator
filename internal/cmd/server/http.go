package server

import (
	"net/http"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type httpServer struct {
	logger hclog.Logger
	addr   string
}

func (h *httpServer) start() {
	if h.logger == nil {
		h.logger = hclog.NewNullLogger()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", h.metricsEndpoint)

	server := &http.Server{
		Addr:    h.addr,
		Handler: mux,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			h.logger.Error(err.Error())
		}
	}()

	h.logger.Info("http server started", "addr", h.addr)
}

var (
	// Only create the prometheus handler once
	promHandler http.Handler
	promOnce    sync.Once
)

func (h *httpServer) metricsEndpoint(w http.ResponseWriter, req *http.Request) {
	handlerOptions := promhttp.HandlerOpts{
		ErrorLog:           h.logger.Named("prometheus_handler").StandardLogger(nil),
		ErrorHandling:      promhttp.ContinueOnError,
		DisableCompression: true,
	}

	promHandler = promhttp.HandlerFor(prometheus.DefaultGatherer, handlerOptions)
	promHandler.ServeHTTP(w, req)
}
