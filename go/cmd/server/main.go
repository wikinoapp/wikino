package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/wikinoapp/wikino/go/internal/handler/health"
)

func main() {
	port := os.Getenv("WIKINO_PORT")
	if port == "" {
		port = "4004"
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	healthHandler := health.NewHandler()

	r.Get("/health", healthHandler.Show)

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	slog.Info("HTTPサーバーを起動します", "addr", addr)

	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("サーバーの起動に失敗しました", "error", err)
		os.Exit(1)
	}
}
