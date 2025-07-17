package crawler

import (
	// Stdlib
	"fmt"
	"math"
	"strings"
	"time"

	// Third-party
	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
)

// initializeKafka sets up the Kafka AsyncProducer with optimized configuration for high-throughput crawling.
// It configures batching, compression, retries, and timeouts for reliable message delivery.
// Returns an error if Kafka connection cannot be established after configured retry attempts.
//
// Configuration highlights:
//   - Uses Snappy compression for better throughput
//   - Batches messages for efficiency (100ms intervals or 100 messages)
//   - Implements exponential backoff retry strategy
//   - Optimizes for local acknowledgment (WaitForLocal) for better performance
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

// sendToKafka sends the scraped HTML content to Kafka asynchronously for downstream processing.
// It performs content size validation, creates a properly structured Kafka message, and attempts
// non-blocking message queuing with a timeout fallback.
//
// Parameters:
//   - url: The source URL of the crawled content (used as message key for partitioning)
//   - html: The scraped HTML content to be sent
//
// Returns:
//   - bool: true if the error is retriable (network issues, channel full), false otherwise
//   - error: any error encountered during the send operation
//
// The function implements the following safeguards:
//   - Content size validation against configured limits
//   - Non-blocking send with timeout to prevent goroutine blocking
//   - Retriable error detection for intelligent retry logic
func (c *Crawler) sendToKafka(item QueueItem, html string) (bool, error) {
	if len(html) > c.Cfg.MaxContentSize {
		return false, fmt.Errorf("HTML content exceeds maximum size of %d bytes", c.Cfg.MaxContentSize)
	}

	// Create the message
	msg := &sarama.ProducerMessage{
		Topic:     c.Cfg.KafkaTopic,
		Key:       sarama.StringEncoder(item.URL),
		Value:     sarama.StringEncoder(html),
		Timestamp: time.Now(),
		Metadata:  item, // Store URL for async error handling
	}

	// Send message asynchronously (non-blocking)
	select {
	case c.AsyncProducer.Input() <- msg:
		// Message queued successfully
		c.Log.WithFields(logrus.Fields{
			"url":   item.URL,
			"depth": item.Depth,
		}).Info("Message sent to Kafka successfully")
		return false, nil
	case <-time.After(100 * time.Millisecond):
		// Channel is full, treat as retriable error
		c.Log.WithField("url", item.URL).Warn("Kafka producer input channel full, requeuing")
		return true, fmt.Errorf("kafka producer channel full")
	}
}

// startAsyncProducerHandlers launches background goroutines to process Kafka AsyncProducer responses.
// This function creates two dedicated handlers that run for the lifetime of the crawler:
//
// Success Handler:
//   - Processes successful message deliveries from Kafka
//   - Updates success metrics for monitoring
//   - Provides debug logging for successful sends
//
// Error Handler:
//   - Processes failed message deliveries and connection errors
//   - Implements intelligent error classification (retriable vs. permanent)
//   - Automatically requeues URLs for retriable errors (network issues)
//   - Updates appropriate error metrics based on error type
//
// Both handlers respect the crawler's shutdown signal and participate in graceful shutdown.
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
					c.Log.WithFields(logrus.Fields{
						"url":   success.Metadata.(QueueItem).URL,
						"depth": success.Metadata.(QueueItem).Depth,
					}).Trace("Message sent to Kafka successfully")
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
				item := err.Msg.Metadata.(QueueItem)
				c.Log.WithFields(map[string]interface{}{
					"url":   item.URL,
					"error": err.Err,
				}).Error("Failed to send message to Kafka")

				// Check if it's a retriable error
				if strings.Contains(err.Err.Error(), "connection refused") ||
					strings.Contains(err.Err.Error(), "no such host") ||
					strings.Contains(err.Err.Error(), "timeout") {
					c.Stats.IncrementKafkaErrored()

					if exists, err := c.isURLRequeued(item.URL); exists {
						c.Log.WithFields(logrus.Fields{"url": item.URL}).Trace("URL already requeued once. Will be marked as visited")
						c.RemoveFromRequeued(item.URL)
					} else {
						// Re-queue URL instead of marking as visited
						c.Log.WithFields(logrus.Fields{"url": item.URL, "error": err}).Warn("Retriable error occurred, requeuing URL")
						c.AddToPending(item)
						c.AddToRequeued(item.URL)
					}
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
