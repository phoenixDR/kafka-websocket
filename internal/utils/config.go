package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Kafka struct {
		Host          string
		Port          string
		Topic         string
		ConsumerGroup string
	}
	Websocket struct {
		Port string
	}
}

func LoadConfig() Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	var cfg Config

	cfg.Kafka.Host = os.Getenv("KAFKA_HOST")
	if cfg.Kafka.Host == "" {
		log.Fatal("KAFKA_HOST not set in .env file")
	}

	cfg.Kafka.Port = os.Getenv("KAFKA_PORT")
	if cfg.Kafka.Port == "" {
		log.Fatal("KAFKA_PORT not set in .env file")
	}

	cfg.Kafka.Topic = os.Getenv("KAFKA_TOPIC")
	if cfg.Kafka.Topic == "" {
		log.Fatal("KAFKA_TOPIC not set in .env file")
	}

	cfg.Kafka.ConsumerGroup = os.Getenv("KAFKA_CONSUMER_GROUP")
	if cfg.Kafka.ConsumerGroup == "" {
		log.Fatal("KAFKA_CONSUMER_GROUP not set in .env file")
	}

	cfg.Websocket.Port = os.Getenv("WEBSOCKET_SERVER_PORT")
	if cfg.Websocket.Port == "" {
		log.Fatal("WEBSOCKET_SERVER_PORT not set in .env file")
	}

	return cfg
}
