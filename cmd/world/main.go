// Command world runs a single Hollow Grid world server (a node on the Grid).
//
// It is fully playable standalone; federation (joining the Grid) is additive.
// See docs/protocol.md and docs/PLAN.md.
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

	"github.com/SkyPhusion/hollow-grid-go/internal/grid"
	"github.com/SkyPhusion/hollow-grid-go/internal/store"
	"github.com/SkyPhusion/hollow-grid-go/internal/transport"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

// applyEnv fills empty flags from the container-friendly env vars (compose, k8s).
func applyEnv(addr, name, url, data, admins, adminToken, gridHubURL, gridHubToken, gridWorldKey *string) {
	if *addr == ":8790" {
		if v := strings.TrimSpace(os.Getenv("LISTEN_ADDR")); v != "" {
			*addr = v
		}
	}
	if *name == "Rust Choir" {
		if v := strings.TrimSpace(os.Getenv("WORLD_NAME")); v != "" {
			*name = v
		}
	}
	if *url == "" {
		*url = strings.TrimSpace(os.Getenv("WORLD_URL"))
	}
	if *data == "data" {
		if v := strings.TrimSpace(os.Getenv("DATA_DIR")); v != "" {
			*data = v
		}
	}
	if *admins == "skyphusion" {
		if v := strings.TrimSpace(os.Getenv("ADMINS")); v != "" {
			*admins = v
		}
	}
	if *gridHubURL == "" {
		*gridHubURL = strings.TrimSpace(os.Getenv("GRID_HUB_URL"))
	}
	if *gridHubToken == "" {
		*gridHubToken = strings.TrimSpace(os.Getenv("GRID_HUB_TOKEN"))
	}
	if *adminToken == "" {
		*adminToken = strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
	}
	if *gridWorldKey == "" {
		*gridWorldKey = strings.TrimSpace(os.Getenv("GRID_WORLD_KEY"))
	}
}

func main() {
	addr := flag.String("addr", ":8790", "listen address")
	name := flag.String("world-name", "Rust Choir", "world display name")
	url := flag.String("world-url", "", "this world's public URL (for the federation registry, e.g. wss://rustchoir.skyphusion.org/ws)")
	data := flag.String("data", "data", "directory for local character persistence")
	admins := flag.String("admins", "skyphusion", "comma-separated keeper names (wall command)")
	adminToken := flag.String("admin-token", "", "shared secret required to log in as a keeper name (ADMIN_TOKEN)")
	gridHubURL := flag.String("grid-hub-url", "", "Grid Hub HTTP RPC URL (e.g. https://grid-hub.example/rpc); omit for standalone LocalHub")
	gridHubToken := flag.String("grid-hub-token", "", "Bearer token for Grid Hub RPC (GRID_RPC_TOKEN)")
	gridWorldKey := flag.String("grid-world-key", "", "Per-world key for mutating Grid Hub RPC (GRID_WORLD_KEY)")
	flag.Parse()
	applyEnv(addr, name, url, data, admins, adminToken, gridHubURL, gridHubToken, gridWorldKey)

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
	var gh grid.Hub
	if strings.TrimSpace(*gridHubURL) != "" {
		gh = grid.NewRemoteHub(*gridHubURL, *gridHubToken, *name, *gridWorldKey)
		log.Info("federation enabled", "grid_hub", *gridHubURL)
	}
	srv := transport.NewServer(w, st, gh, adminList, *adminToken, log)

	fedCtx, fedCancel := context.WithCancel(context.Background())
	defer fedCancel()
	srv.RunWorldLoop(fedCtx)
	srv.RunFederation(fedCtx)

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
