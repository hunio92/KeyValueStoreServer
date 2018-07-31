package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"store"
	"syscall"
	"time"
)

func main() {
	const Host = "127.0.0.1"
	const Port = "8080"
	const MaxKeyValues = 2

	db := store.NewDatabase()
	service := store.NewService(db, MaxKeyValues)

	hs, logger := setup(service, Host, Port)

	go func() {
		logger.Printf("Listening on htt://%s\n", hs.Addr)

		if err := hs.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatal(err)
		}
	}()

	graceful(hs, logger, 5*time.Second)
}

func setup(service *store.Service, host, port string) (*http.Server, *log.Logger) {
	logger := log.New(os.Stdout, "", 0)

	return &http.Server{
		Addr:    host + ":" + port,
		Handler: store.NewServer(service, store.Logger(logger)),
	}, logger
}

func graceful(hs *http.Server, logger *log.Logger, timeout time.Duration) {
	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	logger.Printf("\nShutdown with timeout: %s\n", timeout)

	if err := hs.Shutdown(ctx); err != nil {
		logger.Printf("Error: %v\n", err)
	} else {
		logger.Println("Server stopped")
	}
}
