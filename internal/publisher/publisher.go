package publisher

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/phoenixDR/kafka-websocket/internal/utils"
	"github.com/segmentio/kafka-go"
)

var (
	intervalFlag = flag.Duration("interval", 5*time.Second, "Interval between messages")
)

type Publisher struct {
	cfg           utils.Config
	brokerAddress string
	conn          *kafka.Conn
}

func NewPublisher() (*Publisher, error) {
	p := &Publisher{}
	err := p.Init()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Publisher) Init() error {
	flag.Parse()
	p.cfg = utils.LoadConfig()
	p.brokerAddress = p.cfg.Kafka.Host + ":" + p.cfg.Kafka.Port
	conn, err := p.getKafkaConnection(p.cfg.Kafka.Topic)
	if err != nil {
		return fmt.Errorf("failed to establish initial Kafka connection: %w", err)
	}
	p.conn = conn
	return p.createTopicIfNotExist(p.cfg.Kafka.Topic, 1, 1)
}

func (p *Publisher) PublishDummyMessages(ctx context.Context) {
	ticker := time.NewTicker(*intervalFlag)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Publisher shutting down.")
			return
		case <-ticker.C:
			if utils.KafkaIsHealthy(p.brokerAddress, p.cfg.Kafka.Topic) {
				p.publishMessages()
			} else {
				log.Println("Kafka is not healthy. Skipping message publish.")
			}
		}
	}
}

func (p *Publisher) publishMessages() {
	cryptos := []string{"BTC", "ETH", "LTC"}

	for _, crypto := range cryptos {
		message := fmt.Sprintf("%s: %f", crypto, randomPrice())
		err := p.sendMessage(message)
		if err != nil {
			log.Println("Error sending message:", err)
		} else {
			log.Println("Message sent successfully")
		}

		time.Sleep(1 * time.Second)
	}
}

func (p *Publisher) sendMessage(message string) error {
	maxRetries := 5
	retryInterval := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err := p.conn.WriteMessages(kafka.Message{Value: []byte(message)})
		if err == nil {
			return nil
		}

		if p.isConnectionError(err) {
			p.conn.Close()
			conn, err := p.getKafkaConnection(p.cfg.Kafka.Topic)
			if err != nil {
				return fmt.Errorf("failed to re-establish connection: %w", err)
			}
			p.conn = conn
			continue
		}

		log.Printf("Failed to send message, attempt %d: %v\n", attempt+1, err)
		time.Sleep(retryInterval)
	}

	return fmt.Errorf("failed to send message after %d attempts", maxRetries)
}

func (p *Publisher) isConnectionError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "closed network connection") ||
		strings.Contains(errMsg, "connection reset by peer") {
		return true
	}

	if os.IsTimeout(err) {
		return true
	}

	return false
}

func (p *Publisher) createTopicIfNotExist(topic string, partition, replicationFactor int) error {
	maxRetries := 5
	backoff := time.Second
	maxBackoff := time.Minute

	conn, err := kafka.Dial("tcp", p.brokerAddress)
	if err != nil {
		return fmt.Errorf("failed to dial Kafka: %w", err)
	}
	defer conn.Close()

	for attempt := 0; attempt < maxRetries; attempt++ {
		partitions, err := conn.ReadPartitions(topic)
		if err != nil && !strings.Contains(err.Error(), "no such topic") {
			if strings.Contains(err.Error(), "Leader Not Available") {
				log.Printf("Leader election is not complete, retrying in %v...", backoff)
				time.Sleep(backoff)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			} else {
				return fmt.Errorf("failed to read partitions: %w", err)
			}
		}

		if len(partitions) == 0 {
			controller, err := conn.Controller()
			if err != nil {
				return fmt.Errorf("failed to get Kafka Controller URL: %w", err)
			}

			controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
			if err != nil {
				return fmt.Errorf("failed to dial Kafka Controller: %w", err)
			}

			topicConfigs := []kafka.TopicConfig{
				{
					Topic:             topic,
					NumPartitions:     partition,
					ReplicationFactor: replicationFactor,
				},
			}

			err = controllerConn.CreateTopics(topicConfigs...)
			controllerConn.Close()
			if err != nil {
				return fmt.Errorf("failed to create Kafka topic: %w", err)
			}
		} else {
			log.Printf("Topic %s already exists\n", topic)
			return nil
		}
	}

	return fmt.Errorf("after %d attempts, topic creation failed", maxRetries)
}

func randomPrice() float64 {
	minPrice := 10.00
	maxPrice := 100.00
	return minPrice + rand.Float64()*(maxPrice-minPrice)
}

func (p *Publisher) getKafkaConnection(topic string) (*kafka.Conn, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", p.brokerAddress, topic, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to dial leader: %w", err)
	}
	return conn, nil
}
