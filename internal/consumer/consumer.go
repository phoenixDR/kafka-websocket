package consumer

import (
	"context"
	"errors"
	"fmt"
	"github.com/phoenixDR/kafka-websocket/internal/utils"
	"github.com/segmentio/kafka-go"
	"log"
)

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(brokers []string, topic string, groupID string) (*KafkaConsumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: groupID,
		Topic:   topic,
	})

	if !utils.KafkaIsHealthy(brokers[0], topic) {
		return nil, fmt.Errorf("failed to connect to Kafka")
	}

	return &KafkaConsumer{reader: reader}, nil
}

func (kc *KafkaConsumer) Consume(ctx context.Context, msgChan chan<- string) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Kafka consumer is stopping due to context cancellation")
			return
		default:
			msg, err := kc.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error fetching message: %v\n", err)
				if errors.Is(err, context.Canceled) {
					return
				}
				continue
			}

			msgChan <- string(msg.Value)
		}
	}
}
