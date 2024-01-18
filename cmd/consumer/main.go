package main

import (
	"context"
	"github.com/phoenixDR/kafka-websocket/internal/utils"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phoenixDR/kafka-websocket/internal/consumer"
	"github.com/phoenixDR/kafka-websocket/internal/websocket"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kafkaHost, kafkaPort, topic, consumerGroup, wsServerPort := utils.LoadEnv()
	brokerAddress := kafkaHost + ":" + kafkaPort
	kafkaBrokers := []string{brokerAddress}
	kafkaConsumer, err := consumer.NewKafkaConsumer(kafkaBrokers, topic, consumerGroup)

	if err != nil {
		log.Fatalf("Error initializing Kafka consumer: %v", err)
	}

	go func() {
		retryCount := 0
		maxRetries := 5
		for {
			if utils.KafkaIsHealthy(brokerAddress, topic) {
				retryCount = 0
			} else {
				retryCount++
				if retryCount >= maxRetries {
					log.Println("Max retries reached, shutting down WebSocket server.")
					websocket.NotifyAndDisconnectClients()
					cancel()
					break
				}
				backoffDuration := utils.ExponentialBackoff(retryCount)
				log.Printf("Kafka health check failed. Retrying in %v...\n", backoffDuration)
				time.Sleep(backoffDuration)
			}
		}
	}()

	msgChan := make(chan string)
	go kafkaConsumer.Consume(ctx, msgChan)
	go websocket.StartWebSocketServer(":" + wsServerPort)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgChan:
				websocket.BroadcastToSubscribedClients(msg)
			}
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-shutdownChan:
		log.Println("OS shutdown signal received")
		cancel()
	case <-ctx.Done():
		log.Println("Shutting down due to health check failure")
	}

	log.Println("Shutting down consumer microservice")
}
