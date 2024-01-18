package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() (kafkaHost, kafkaPort, topic, consumerGroup, wsServerPort string) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	kafkaHost = os.Getenv("KAFKA_HOST")
	if kafkaHost == "" {
		log.Fatal("KAFKA_HOST not set in .env file")
	}

	kafkaPort = os.Getenv("KAFKA_PORT")
	if kafkaPort == "" {
		log.Fatal("KAFKA_PORT not set in .env file")
	}

	topic = os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		log.Fatal("KAFKA_TOPIC not set in .env file")
	}

	consumerGroup = os.Getenv("KAFKA_CONSUMER_GROUP")
	if consumerGroup == "" {
		log.Fatal("KAFKA_CONSUMER_GROUP not set in .env file")
	}

	wsServerPort = os.Getenv("WEBSOCKET_SERVER_PORT")
	if wsServerPort == "" {
		log.Fatal("WEBSOCKET_SERVER_PORT not set in .env file")
	}

	return kafkaHost, kafkaPort, topic, consumerGroup, wsServerPort
}
