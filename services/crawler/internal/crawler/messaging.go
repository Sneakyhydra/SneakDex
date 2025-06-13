package crawler

import (
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
)

// initializeKafka sets up the Sarama AsyncProducer.
// It configures producer properties and starts goroutines to handle success and error callbacks.
func (crawler *Crawler) initializeKafka() error {
	cfg := config.GetConfig() // Assuming kafka topic is part of config

	kafkaConfig := sarama.NewConfig()
	// Crucial for AsyncProducer: enable return channels for success and errors
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true

	// RequiredAcks specifies the number of acknowledgements the producer requires
	// from the broker before considering a request complete.
	// sarama.WaitForAll: will wait for all in-sync replicas to commit the message. (Highest durability)
	// sarama.WaitForLocal: will wait for the leader to write the record to its local log. (Good balance)
	// sarama.NoResponse: producer will not wait for any acknowledgment. (Highest throughput, lowest durability)
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll // Or adjust based on your durability needs

	// Configure retries for transient errors
	kafkaConfig.Producer.Retry.Max = cfg.KafkaRetryMax
	kafkaConfig.Producer.Retry.Backoff = 250 * time.Millisecond // Time to wait before retrying

	// Configure batching for throughput optimization
	// Messages are batched before sending.
	// Flush.Bytes: Max bytes to buffer before sending a batch.
	// Flush.Messages: Max messages to buffer before sending a batch.
	// Flush.Frequency: Max time to wait before sending a batch, even if not full.
	kafkaConfig.Producer.Flush.Bytes = cfg.MaxContentSize         // 5MB batch size
	kafkaConfig.Producer.Flush.Messages = 1000                    // Flush after 1000 messages
	kafkaConfig.Producer.Flush.Frequency = 100 * time.Millisecond // Flush every 100ms
	kafkaConfig.Producer.MaxMessageBytes = cfg.MaxContentSize

	// Optional: Add compression to reduce network traffic
	kafkaConfig.Producer.Compression = sarama.CompressionSnappy

	producer, err := sarama.NewAsyncProducer(strings.Split(cfg.KafkaBrokers, ","), kafkaConfig)
	if err != nil {
		return fmt.Errorf("failed to create Sarama AsyncProducer: %w", err)
	}
	crawler.producer = producer

	// Start goroutines to handle success and error callbacks from the producer.
	// These are essential for the AsyncProducer to function correctly and avoid deadlocks.
	wg.Add(2) // Add to WaitGroup for these new goroutines
	go crawler.handleKafkaSuccesses()
	go crawler.handleKafkaErrors()

	return nil
}

// handleKafkaSuccesses continuously reads from the Kafka producer's Successes channel.
// It logs successful message deliveries, updates Kafka-specific metrics, and
// increments the overall pages successful counter, tying it to confirmed Kafka delivery.
// This goroutine runs until the shutdown signal is received or context is cancelled.
func (crawler *Crawler) handleKafkaSuccesses() {
	log := logger.GetLogger()
	defer wg.Done() // Decrement WaitGroup when this goroutine exits.

	log.Info("Starting Kafka success handler goroutine.")
	for {
		select {
		case msg, ok := <-crawler.producer.Successes():
			if !ok { // Channel closed, producer is shutting down or has already closed.
				log.Info("Kafka producer successes channel closed. Stopping success handler.")
				return
			}

			crawler.stats.IncrementKafkaSuccessful()

			// Safely get string key from sarama.Encoder interface for logging
			keyStr := ""
			if msg.Key != nil {
				if encodedKey, err := msg.Key.Encode(); err == nil {
					keyStr = string(encodedKey)
				} else {
					log.WithError(err).Warn("Failed to encode Kafka message key for logging in success handler.")
				}
			}

			// Use Debug level for individual message successes to avoid excessive log volume.
			// Consider changing this to Info if you want to see every successful page in your main logs.
			log.WithFields(logrus.Fields{
				"topic":     msg.Topic,
				"partition": msg.Partition,
				"offset":    msg.Offset,
				"key":       keyStr, // Use the safely extracted string key
				"timestamp": msg.Timestamp.Format(time.RFC3339),
			}).Debug("Kafka message successfully produced and page counted as successful.")
		case <-shutdown:
			log.Info("Stopping Kafka success handler goroutine via shutdown signal.")
			return
		case <-ctx.Done():
			log.Info("Stopping Kafka success handler goroutine via context cancellation.")
			return
		}
	}
}

// handleKafkaErrors continuously reads from the Kafka producer's Errors channel.
// It logs failed message deliveries, updates metrics, and implements retry logic
// by re-queueing URLs to Redis for transient Kafka production failures.
// This goroutine runs until the shutdown signal is received or context is cancelled.
func (crawler *Crawler) handleKafkaErrors() {
	log := logger.GetLogger()
	defer wg.Done() // Decrement WaitGroup when this goroutine exits.

	log.Info("Starting Kafka error handler goroutine.")
	for {
		select {
		case err, ok := <-crawler.producer.Errors():
			if !ok { // Channel closed, producer is shutting down or has already closed.
				log.Info("Kafka producer errors channel closed. Stopping error handler.")
				return
			}

			crawler.stats.IncrementKafkaFailed()

			// Safely get string key (which is the URL) from sarama.Encoder interface for the failed message.
			urlToRequeue := ""
			if err.Msg != nil && err.Msg.Key != nil {
				if encodedKey, encodeErr := err.Msg.Key.Encode(); encodeErr == nil {
					urlToRequeue = string(encodedKey)
				} else {
					log.WithError(encodeErr).Warn("Failed to encode Kafka error message key for logging; cannot re-queue.")
				}
			}

			logFields := logrus.Fields{
				"topic": err.Msg.Topic,
				"url":   urlToRequeue, // Use the extracted URL
				"error": err.Error(),
			}

			// Log the Kafka production failure.
			log.WithFields(logFields).Error("Failed to produce Kafka message.")

			// --- Re-queueing Logic ---
			if urlToRequeue != "" {
				// Check if this URL has been re-queued before in this crawler instance.
				// This simple check helps prevent immediate infinite loops.
				// For more robust retry limits, consider storing a retry count in Redis.
				if _, loaded := crawler.requeued.LoadOrStore(urlToRequeue, struct{}{}); loaded {
					// If it was already in 'requeued' map, it means we tried to re-queue it.
					// This indicates a persistent problem or a high-frequency retry scenario.
					// We log it but do NOT re-queue again from this instance for now.
					log.WithFields(logFields).Warn("URL previously re-queued due to Kafka error. Skipping further re-queueing attempts from this instance to prevent infinite loops.")
					crawler.stats.IncrementRedisFailed() // Count as a failure to re-queue effectively
					continue                             // Move to the next error
				}

				// Attempt to re-queue the URL back to Redis 'pending_urls' set.
				_, redisErr := crawler.redisClient.SAdd(ctx, "crawler:pending_urls", urlToRequeue).Result()
				if redisErr != nil {
					log.WithFields(logFields).WithError(redisErr).Error("Failed to re-queue URL to Redis 'pending_urls' after Kafka production failure.")
					crawler.stats.IncrementRedisFailed()
				} else {
					log.WithFields(logFields).Info("Successfully re-queued URL to Redis 'pending_urls' after Kafka production failure.")
					crawler.stats.IncrementRedisSuccessful() // Or a specific 'requeue_successful' metric
				}
			} else {
				log.WithFields(logFields).Warn("Kafka message key (URL) was empty or unreadable; cannot re-queue.")
			}

		case <-shutdown:
			log.Info("Stopping Kafka error handler goroutine via shutdown signal.")
			return
		case <-ctx.Done():
			log.Info("Stopping Kafka error handler goroutine via context cancellation.")
			return
		}
	}
}

// publishPageData sends the crawled page data to Kafka asynchronously.
// It constructs a Sarama ProducerMessage and sends it to the producer's input channel.
// This method is non-blocking to the caller (Colly's OnHTML/OnRequest callbacks).
// It handles potential backpressure by logging a warning if the Kafka input channel is full.
func (crawler *Crawler) publishPageData(data []byte, url string) {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	msg := &sarama.ProducerMessage{
		Topic: cfg.KafkaTopic, // Use topic from configuration
		Value: sarama.ByteEncoder(data),
		Key:   sarama.StringEncoder(url), // Using URL as key can ensure messages for the same URL go to the same partition
	}

	// Non-blocking send to the input channel using a select statement.
	// If the channel is full, the 'default' case will execute immediately.
	select {
	case crawler.producer.Input() <- msg:
		log.WithField("url", url).Debug("Enqueued Kafka message for asynchronous production.")
		// The actual success/failure will be handled by handleKafkaSuccesses/Errors goroutines.
	case <-ctx.Done():
		// Context cancelled, producer might be shutting down or already closed.
		log.WithField("url", url).Warn("Context cancelled, unable to enqueue Kafka message.")
		crawler.stats.IncrementKafkaFailed() // Consider this a failure for immediate metrics.
	default:
		// This branch executes if the producer.Input() channel is full.
		// This indicates backpressure from Kafka or a slower consumer.
		log.WithField("url", url).Warn("Kafka producer input channel full, message dropped to avoid blocking. Consider increasing Kafka throughput or producer buffer.")
		crawler.stats.IncrementKafkaFailed() // Count as a dropped/failed message
	}
}
