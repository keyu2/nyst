package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keyu2/nyst/server"
	"github.com/keyu2/nyst/server/cache"
	"github.com/keyu2/nyst/server/mta"
)

//go:embed all:web/dist
var webFiles embed.FS

func main() {
	dist, err := fs.Sub(webFiles, "web/dist")
	if err != nil {
		slog.Error("open embedded web app", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store := cache.NewStore()
	client := mta.NewClient()
	realtime := mta.NewRealtimeService()
	static := mta.NewStaticService(realtime)
	metadata := mta.NewMetadataService()
	transit := mta.NewTransitService(static, realtime, metadata)
	go client.RefreshAlerts(ctx, store, 30*time.Second)

	app := server.New(dist, store, static, transit, realtime, client)
	if err := app.Run(ctx); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
