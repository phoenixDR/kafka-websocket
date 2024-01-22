package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/phoenixDR/kafka-websocket/internal/publisher"
)

func main() {
	p, err := publisher.NewPublisher()
	if err != nil {
		log.Fatal("Error initializing publisher:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.PublishDummyMessages(ctx)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-signalChan:
		log.Println("Received shutdown signal")
		cancel()
	case <-ctx.Done():
	}

	log.Println("Publisher has been stopped")
}
