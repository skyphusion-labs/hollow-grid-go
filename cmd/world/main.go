// Command world runs a single Hollow Grid world server (a node on the Grid).
//
// It is fully playable standalone; federation (joining the Grid) is additive and
// arrives in a later phase. See docs/protocol.md and docs/PLAN.md.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/store"
	"github.com/SkyPhusion/hollow-grid-go/internal/transport"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

func main() {
	addr := flag.String("addr", ":8790", "listen address")
	name := flag.String("world-name", "Rust Choir", "world display name")
	url := flag.String("world-url", "", "this world's public URL (for the federation registry, e.g. wss://rustchoir.skyphusion.org/ws)")
	data := flag.String("data", "data", "directory for local character persistence")
	admins := flag.String("admins", "skyphusion", "comma-separated keeper names (wall command)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	st, err := store.NewFileStore(*data)
	if err != nil {
		log.Error("character store failed", "dir", *data, "err", err)
		os.Exit(1)
	}

	w := world.New(*name, *url)
	var adminList []string
	for _, a := range strings.Split(*admins, ",") {
		if t := strings.TrimSpace(a); t != "" {
			adminList = append(adminList, t)
		}
	}
	srv := transport.NewServer(w, st, nil, adminList, log)

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("hollow-grid-go listening", "addr", *addr, "world", *name, "data", *data)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutCtx)
}
