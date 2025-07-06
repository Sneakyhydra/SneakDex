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

	// Set Kafka producer configurations for AsyncProducer
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForLocal // Faster than WaitForAll
	kafkaConfig.Producer.Retry.Max = c.Cfg.KafkaRetryMax
	kafkaConfig.Producer.Retry.Backoff = 100 * time.Millisecond
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	kafkaConfig.Producer.Timeout = c.Cfg.RequestTimeout
	kafkaConfig.Net.DialTimeout = c.Cfg.RequestTimeout
	kafkaConfig.Metadata.RefreshFrequency = 10 * time.Minute
	kafkaConfig.Producer.MaxMessageBytes = c.Cfg.MaxContentSize
	kafkaConfig.Producer.Compression = sarama.CompressionSnappy
	// Async producer optimizations
	kafkaConfig.Producer.Flush.Frequency = 100 * time.Millisecond
	kafkaConfig.Producer.Flush.Messages = 100
	kafkaConfig.Producer.Flush.Bytes = 1024 * 1024 // 1MB

	// Create a new Kafka producer
	brokers := strings.Split(c.Cfg.KafkaBrokers, ",")

	for attempt := 1; attempt <= c.Cfg.KafkaRetryMax; attempt++ {
		producer, err := sarama.NewAsyncProducer(brokers, kafkaConfig)
		if err == nil {
			c.AsyncProducer = producer
			c.Log.Info("Kafka AsyncProducer initialized")
			// Start async producer handlers
			c.startAsyncProducerHandlers()
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

// sendToKafka sends the scraped HTML content to Kafka asynchronously.
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
		Metadata:  url, // Store URL for async error handling
	}

	// Send message asynchronously (non-blocking)
	select {
	case c.AsyncProducer.Input() <- msg:
		// Message queued successfully
		return false, nil
	case <-time.After(100 * time.Millisecond):
		// Channel is full, treat as retriable error
		c.Log.WithField("url", url).Warn("Kafka producer input channel full, requeuing")
		return true, fmt.Errorf("kafka producer channel full")
	}
}

// startAsyncProducerHandlers starts goroutines to handle async producer success/error messages
func (c *Crawler) startAsyncProducerHandlers() {
	c.Wg.Add(2)
	
	// Success handler
	go func() {
		defer c.Wg.Done()
		for {
			select {
			case success := <-c.AsyncProducer.Successes():
				c.Stats.IncrementKafkaSuccessful()
				if c.Cfg.EnableDebug {
					c.Log.WithField("url", success.Metadata).Trace("Message sent to Kafka successfully")
				}
			case <-c.CShutdown:
				c.Log.Info("Stopping Kafka success handler")
				return
			}
		}
	}()
	
	// Error handler
	go func() {
		defer c.Wg.Done()
		for {
			select {
			case err := <-c.AsyncProducer.Errors():
				url := err.Msg.Metadata.(string)
				c.Log.WithFields(map[string]interface{}{
					"url":   url,
					"error": err.Err,
				}).Error("Failed to send message to Kafka")
				
				// Check if it's a retriable error
				if strings.Contains(err.Err.Error(), "connection refused") ||
					strings.Contains(err.Err.Error(), "no such host") ||
					strings.Contains(err.Err.Error(), "timeout") {
					c.Stats.IncrementKafkaErrored()
					// Optionally requeue the URL
					c.AddToPending(url)
				} else {
					c.Stats.IncrementKafkaFailed()
				}
			case <-c.CShutdown:
				c.Log.Info("Stopping Kafka error handler")
				return
			}
		}
	}()
}
