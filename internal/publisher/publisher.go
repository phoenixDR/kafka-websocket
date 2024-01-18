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

func PublishDummyMessages(ctx context.Context) {
	flag.Parse()

	kafkaHost, kafkaPort, topic, _, _ := utils.LoadEnv()
	brokerAddress := kafkaHost + ":" + kafkaPort
	rand.Seed(time.Now().UnixNano())

	err := createTopicIfNotExist(brokerAddress, topic, 1, 1)
	if err != nil {
		log.Fatal("Creation topic process got error:", err)
	}

	conn, err := getKafkaConnection(nil, brokerAddress, topic)
	if err != nil {
		log.Fatal("Failed to set up initial Kafka connection:", err)
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	ticker := time.NewTicker(*intervalFlag)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Publisher shutting down.")
			return
		case <-ticker.C:
			if utils.KafkaIsHealthy(brokerAddress, topic) {
				publishMessages(&conn, brokerAddress, topic)
			} else {
				log.Println("Kafka is not healthy. Skipping message publish.")
			}
		}
	}
}

func publishMessages(conn **kafka.Conn, brokerAddress, topic string) {
	cryptos := []string{"BTC", "ETH", "LTC"}
	var err error

	for _, crypto := range cryptos {
		message := fmt.Sprintf("%s: %f", crypto, randomPrice())
		*conn, err = sendMessage(*conn, message, brokerAddress, topic)
		if err != nil {
			log.Println("Error sending message:", err)
			if *conn != nil {
				(*conn).Close()
				*conn = nil
			}
		} else {
			log.Println("Message sent successfully")
		}

		time.Sleep(1 * time.Second)
	}
}

func getKafkaConnection(currentConn *kafka.Conn, brokerAddress, topic string) (*kafka.Conn, error) {
	if currentConn != nil {
		return currentConn, nil
	}

	conn, err := kafka.DialLeader(context.Background(), "tcp", brokerAddress, topic, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to dial leader: %w", err)
	}

	return conn, nil
}

func sendMessage(conn *kafka.Conn, message string, brokerAddress, topic string) (*kafka.Conn, error) {
	maxRetries := 5
	retryInterval := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err := conn.WriteMessages(kafka.Message{Value: []byte(message)})
		if err == nil {
			return conn, nil
		}

		if isConnectionError(err) {
			conn.Close()
			conn, err = getKafkaConnection(nil, brokerAddress, topic)
			if err != nil {
				return nil, fmt.Errorf("failed to re-establish connection: %w", err)
			}

			continue
		}

		log.Printf("Failed to send message, attempt %d: %v\n", attempt+1, err)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("failed to send message after %d attempts", maxRetries)
}

func isConnectionError(err error) bool {
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

func createTopicIfNotExist(brokerAddress, topic string, partition, replicationFactor int) error {
	maxRetries := 5
	backoff := time.Second
	maxBackoff := time.Minute

	conn, err := kafka.Dial("tcp", brokerAddress)
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
