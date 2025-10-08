package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-hclog"

	"github.com/peteski22/plugins-demo/internal/plugins"
	"github.com/peteski22/plugins-demo/internal/plugins/pipeline"
	pkg "github.com/peteski22/plugins-demo/pkg/contract/plugin"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func pluginPaths() ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return []string{filepath.Join(cwd, "bin", "sample-plugins")}, nil
}

func pluginBinaries(dirs []string) ([]string, error) {
	var results []string

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("reading directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())

			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
			}

			if info.Mode()&0o111 != 0 {
				results = append(results, path)
			}
		}
	}

	return results, nil
}

func run() error {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "plugins-demo",
		Level: hclog.Debug,
	})

	logger.Info("starting plugins-demo")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	paths, err := pluginPaths()
	if err != nil {
		return fmt.Errorf("error gathering plugin paths: %w", err)
	}

	binaries, err := pluginBinaries(paths)
	if err != nil {
		return fmt.Errorf("error gathering plugin binaries: %w", err)
	}

	logger.Info("found plugin binaries", "count", len(binaries))

	manager := plugins.NewManager(logger)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := manager.StopAll(shutdownCtx); err != nil {
			logger.Error("failed to stop plugins", "error", err)
		}
	}()

	p := pipeline.NewPipeline(logger)

	for _, binary := range binaries {
		instance, err := manager.Start(ctx, binary)
		if err != nil {
			logger.Error("failed to start plugin", "path", binary, "error", err)
			continue
		}

		// Categorize plugins based on their name.
		var category pkg.Category
		name := filepath.Base(binary)
		switch {
		case strings.Contains(name, "header-transformer"):
			category = pkg.CategoryContent
		case strings.Contains(name, "prompt-guard"):
			category = pkg.CategoryValidation
		case strings.Contains(name, "rate-limit"):
			category = pkg.CategoryRateLimiting
		case strings.Contains(name, "tool-audit"):
			category = pkg.CategoryObservability
		default:
			category = pkg.CategoryValidation
		}

		p.Register(category, instance)
		logger.Info("registered plugin", "id", instance.ID(), "category", category)
	}

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)

	router.Use(p.Middleware())

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	router.Get("/api/v1/example", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"message":"Hello from plugins-demo","time":"%s"}`, time.Now().Format(time.RFC3339))
	})

	router.Post("/api/v1/echo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"echo":"received","time":"%s"}`, time.Now().Format(time.RFC3339))
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting down server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown error", "error", err)
		}
	}()

	logger.Info("server starting", "addr", ":8080")

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
