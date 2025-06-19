package crawler

import (
	// Stdlib
	"fmt"
	"math"
	"strings"
	"time"

	// Third-party
	"github.com/IBM/sarama"
)

// initializeKafka sets up Kafka producer with proper configuration
func (c *Crawler) initializeKafka() error {
	//  Create a new Sarama configuration
	kafkaConfig := sarama.NewConfig()

	// Set Kafka producer configurations
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Retry.Max = c.Cfg.KafkaRetryMax
	kafkaConfig.Producer.Retry.Backoff = 100 * time.Millisecond
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	kafkaConfig.Producer.Timeout = c.Cfg.RequestTimeout
	kafkaConfig.Net.DialTimeout = c.Cfg.RequestTimeout
	kafkaConfig.Metadata.RefreshFrequency = 10 * time.Minute
	kafkaConfig.Producer.MaxMessageBytes = c.Cfg.MaxContentSize
	kafkaConfig.Producer.Compression = sarama.CompressionSnappy

	// Create a new Kafka producer
	brokers := strings.Split(c.Cfg.KafkaBrokers, ",")

	for attempt := 1; attempt <= c.Cfg.KafkaRetryMax; attempt++ {
		producer, err := sarama.NewSyncProducer(brokers, kafkaConfig)
		if err == nil {
			c.Producer = producer
			c.Log.Info("Kafka producer initialized")
			return nil
		}
		c.Log.Warnf("Kafka producer initialization attempt %d/%d failed: %v", attempt, c.Cfg.KafkaRetryMax, err)
		if attempt < c.Cfg.KafkaRetryMax {
			// Exponential backoff for retries
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed to create kafka producer after %d attempts. please ensure kafka is running on %s", c.Cfg.KafkaRetryMax, strings.Join(brokers, ","))
}

// sendToKafka sends the scraped HTML content to Kafka.
// Returns true if the url is retriable.
func (c *Crawler) sendToKafka(url, html string) (bool, error) {
	if len(html) > c.Cfg.MaxContentSize {
		return false, fmt.Errorf("HTML content exceeds maximum size of %d bytes", c.Cfg.MaxContentSize)
	}

	// Create the message
	msg := &sarama.ProducerMessage{
		Topic:     c.Cfg.KafkaTopic,
		Key:       sarama.StringEncoder(url),
		Value:     sarama.StringEncoder(html),
		Timestamp: time.Now(),
	}

	var lastErr error
	for attempt := 1; attempt <= c.Cfg.KafkaRetryMax; attempt++ {
		_, _, err := c.Producer.SendMessage(msg)
		if err == nil {
			c.Stats.IncrementKafkaSuccessful()
			return false, nil
		}

		lastErr = err
		c.Log.Warnf("Kafka SendMessage attempt %d/%d failed for URL %s: %v",
			attempt, c.Cfg.KafkaRetryMax, url, err)

		// Don't sleep after the last attempt
		if attempt < c.Cfg.KafkaRetryMax {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// I want to increment kafka errored only when the error is connection related
	if strings.Contains(lastErr.Error(), "connection refused") ||
		strings.Contains(lastErr.Error(), "no such host") ||
		strings.Contains(lastErr.Error(), "timeout") {
		c.Stats.IncrementKafkaErrored()
	} else {
		// For other errors, increment the failed counter
		c.Stats.IncrementKafkaFailed()
	}

	return true, fmt.Errorf("failed to send message to Kafka after %d attempts: %w", c.Cfg.KafkaRetryMax, lastErr)
}
