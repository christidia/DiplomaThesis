package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"load-balancer/config"
	"load-balancer/db"
	"load-balancer/weights"

	"github.com/parnurzeal/gorequest"
	"github.com/streadway/amqp"
)

type Queue struct {
	Name     string `json:"name"`
	Messages int    `json:"messages"`
}

func SetupRabbitMQ() (*amqp.Connection, *amqp.Channel, error) {
	log.Println("🔌 Setting up RabbitMQ connection")
	conn, err := amqp.Dial(config.RabbitMQURL)
	if err != nil {
		return nil, nil, fmt.Errorf("❌ failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("❌ failed to open a channel: %v", err)
	}

	log.Println("✅ RabbitMQ connection and channel set up successfully")
	return conn, ch, nil
}

func FindQueueWithPrefix(prefix string) (string, error) {
	request := gorequest.New()
	apiURL := `http://rabbitmq.rabbitmq-setup.svc.cluster.local:15672/api/queues`
	resp, body, errs := request.Get(apiURL).
		SetBasicAuth(config.RabbitMQUser, config.RabbitMQPass).
		End()

	if len(errs) > 0 {
		return "", fmt.Errorf("❌ Failed to get queues: %v", errs)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("❌ Unexpected status code: %d", resp.StatusCode)
	}

	var queues []Queue
	err := json.Unmarshal([]byte(body), &queues)
	if err != nil {
		return "", fmt.Errorf("❌ Failed to parse response: %v", err)
	}

	for _, queue := range queues {
		if strings.HasPrefix(queue.Name, prefix) {
			return queue.Name, nil
		}
	}

	return "", nil // Queue with the specified prefix not found
}

func PollQueue(queueName string, ch *amqp.Channel, done chan bool) {
	log.Printf("📡 Starting to poll queue %s", queueName)
	for {
		select {
		case <-done:
			log.Println("🛑 Stopping pollQueue goroutine")
			return
		default:
			messageCount, err := CheckQueue(queueName, ch)
			if err != nil {
				log.Printf("❌ Error checking queue: %v\n", err)
			} else {
				log.Printf("📋 Queue %s has %d messages\n", queueName, messageCount)
				if messageCount == 0 {
					//log.Printf("📭 Queue %s is now empty\n", queueName)
					weights.UpdateEmptyQWeightRoutine()
				} else if messageCount > 0 {
					db.PrevQueueEmpty = false
				}
			}
			time.Sleep(config.CheckInterval)
		}
	}
}

func CheckQueue(queueName string, ch *amqp.Channel) (int, error) {
	log.Printf("🔍 Checking queue: %s", queueName)
	queue, err := ch.QueueInspect(queueName)
	if err != nil {
		return 0, fmt.Errorf("❌ failed to inspect queue: %v", err)
	}
	return queue.Messages, nil
}
