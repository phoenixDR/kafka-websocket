package utils

import (
	"context"
	"math"
	"time"

	"github.com/segmentio/kafka-go"
)

func KafkaIsHealthy(brokerAddress, topic string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokerAddress},
		Topic:   topic,
	})
	_, err := reader.Config().Dialer.DialLeader(ctx, "tcp", brokerAddress, topic, 0)
	if err != nil {
		return false
	}

	return true
}

func ExponentialBackoff(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}
