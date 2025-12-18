package main

import (
	"context"
	"flag"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"

	"meow-ai/config"
	"meow-ai/server"
)

func main() {
	_ = flag.Set("logtostderr", "true")
	flag.Parse()

	cfg := config.MustLoad("config.yaml")
	handler := server.NewHandler(cfg)

	mux := http.NewServeMux()
	handler.Register(mux)

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			glog.Warningf("server shutdown error: %v", err)
		}
	}()

	glog.Infof("server listening on %s", cfg.Addr())
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		glog.Fatalf("server error: %v", err)
	}
}
