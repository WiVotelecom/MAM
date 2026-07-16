// Command netinsight starts the NetInsight diagnostics server.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WiVotelecom/MAM/internal/server"
	"github.com/WiVotelecom/MAM/internal/store"
	"github.com/WiVotelecom/MAM/web"
)

func main() {
	addr := flag.String("addr", envOr("NETINSIGHT_ADDR", ":8080"), "listen address")
	dbPath := flag.String("db", envOr("NETINSIGHT_DB", "/data/netinsight.db"), "SQLite database path")
	// A present-but-empty NETINSIGHT_PUBLIC_IP_URL disables the lookup, which is
	// the desired behaviour for air-gapped deployments.
	defaultPublicIPURL := "https://api.ipify.org"
	if v, ok := os.LookupEnv("NETINSIGHT_PUBLIC_IP_URL"); ok {
		defaultPublicIPURL = v
	}
	publicIPURL := flag.String("public-ip-url", defaultPublicIPURL,
		"URL used to discover the public IP (empty to disable for air-gap)")
	flag.Parse()

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	assets, err := web.Dist()
	if err != nil {
		log.Fatalf("load embedded assets: %v", err)
	}

	cfg := server.DefaultConfig()
	cfg.PublicIPURL = *publicIPURL
	srv := server.New(st, assets, cfg)

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("NetInsight %s listening on %s", server.Version, *addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("NetInsight stopped")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
