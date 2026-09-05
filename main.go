package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-upwork-job/internal/api"
	"go-upwork-job/internal/config"
	"go-upwork-job/internal/migrate"
	"go-upwork-job/internal/notifier"
	"go-upwork-job/internal/store"
)

func main() {
	migrateCmd := flag.String("migrate", "", "run migration (up, down, status)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	st := store.New(db)
	runner := migrate.New(db, "db/migrate")

	switch *migrateCmd {
	case "up":
		if err := runner.Up(); err != nil {
			log.Fatalf("migrate up: %v", err)
		}
		return
	case "down":
		if err := runner.Down(); err != nil {
			log.Fatalf("migrate down: %v", err)
		}
		return
	case "status":
		if err := runner.Status(); err != nil {
			log.Fatalf("migrate status: %v", err)
		}
		return
	case "":
	default:
		log.Fatalf("unknown -migrate value %q (use up, down, status)", *migrateCmd)
	}

	nf := notifier.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom, cfg.NotifyEmail)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: api.New(st, nf, cfg.APIKey).Routes(),
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("server stopped")
}