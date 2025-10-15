package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"moph-ic-proxy/internal/cache"
	"moph-ic-proxy/internal/config"
	"moph-ic-proxy/internal/httpclient"
	"moph-ic-proxy/internal/keygen"
	"moph-ic-proxy/internal/middleware"
	"moph-ic-proxy/internal/server"
	"moph-ic-proxy/internal/templates"
	"moph-ic-proxy/internal/web"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("fatal configuration error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	redisClient, err := cache.Connect(ctx, cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		logger.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisClient.Close()
	logger.Println("INFO Redis connected")

	tokenManager := httpclient.NewTokenManager(cfg, redisClient)

	keyManager := keygen.NewManager()
	if cfg.UseAPIKey {
		if err := keyManager.Init(); err != nil {
			logger.Fatalf("failed to initialise API key manager: %v", err)
		}
		logger.Println("INFO API key support enabled")
	}

	tpl, err := templates.Shared()
	if err != nil {
		logger.Fatalf("failed to parse templates: %v", err)
	}

	clientManager, err := httpclient.NewManager(cfg, tokenManager)
	if err != nil {
		logger.Fatalf("failed to create HTTP clients: %v", err)
	}

	webServer := web.NewServer(cfg, tpl, tokenManager, redisClient, keyManager, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	webServer.Register(mux)

	proxy := server.NewProxyHandler(clientManager, logger)
	apiHandler := middleware.APIKeyVerifier(cfg, keyManager, logger)(http.StripPrefix("/api", proxy))
	mux.Handle("/api/", apiHandler)
	mux.Handle("/api", apiHandler)

	go warmTokens(context.Background(), tokenManager, logger)

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      loggingMiddleware(logger)(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Printf("INFO starting server on %s", cfg.Addr())
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("server error: %v", err)
	}
}

func warmTokens(ctx context.Context, manager *httpclient.TokenManager, logger *log.Logger) {
	apps := []string{"mophic", "fdh"}
	for _, app := range apps {
		if _, err := manager.GetToken(ctx, httpclient.GetTokenOptions{Force: true, App: app}); err != nil {
			logger.Printf("WARN unable to prefetch %s token: %v", app, err)
		} else {
			logger.Printf("INFO prefetched %s token", app)
		}
	}
}

func loggingMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			duration := time.Since(start)
			logger.Printf("INFO %s %s status=%d duration=%s", r.Method, r.URL.String(), recorder.status, duration)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
