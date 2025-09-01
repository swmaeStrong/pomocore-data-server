package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"pomocore-data/shared/common/logger"
)

type StreamConfig struct {
	StreamKey string
	Group     string
	Consumer  string
}

type MessageProcessor interface {
	ProcessBatch(ctx context.Context, messages []redis.XMessage) error
}

type AbstractConsumer struct {
	client     *redis.Client
	config     StreamConfig
	processor  MessageProcessor
	workerPool int
	batchSize  int
	blockTime  time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func NewAbstractConsumer(
	client *redis.Client,
	config StreamConfig,
	processor MessageProcessor,
	workerPool int,
	batchSize int,
	blockTime time.Duration,
) *AbstractConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	return &AbstractConsumer{
		client:     client,
		config:     config,
		processor:  processor,
		workerPool: workerPool,
		batchSize:  batchSize,
		blockTime:  blockTime,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (c *AbstractConsumer) Start() error {
	err := c.createConsumerGroup()
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	batchChan := make(chan []redis.XMessage, c.workerPool*2)

	for i := 0; i < c.workerPool; i++ {
		c.wg.Add(1)
		go c.batchWorker(i, batchChan)
	}

	c.wg.Add(1)
	go c.consume(batchChan)

	logger.Info("Consumer started",
		zap.String("stream", c.config.StreamKey),
		zap.Int("workers", c.workerPool))
	return nil
}

func (c *AbstractConsumer) Stop() {
	logger.Info("Stopping consumer", zap.String("stream", c.config.StreamKey))
	c.cancel()
	c.wg.Wait()
	logger.Info("Consumer stopped", zap.String("stream", c.config.StreamKey))
}

func (c *AbstractConsumer) createConsumerGroup() error {
	_, err := c.client.XGroupCreateMkStream(
		c.ctx,
		c.config.StreamKey,
		c.config.Group,
		"0",
	).Result()

	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (c *AbstractConsumer) consume(batchChan chan<- []redis.XMessage) {
	defer c.wg.Done()
	defer close(batchChan)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			messages, err := c.client.XReadGroup(c.ctx, &redis.XReadGroupArgs{
				Group:    c.config.Group,
				Consumer: c.config.Consumer,
				Streams:  []string{c.config.StreamKey, ">"},
				Count:    int64(c.batchSize),
				Block:    c.blockTime,
			}).Result()

			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue
				}
				logger.Error("Error reading from stream",
					zap.String("stream", c.config.StreamKey),
					logger.WithError(err))
				time.Sleep(3 * time.Second)
				continue
			}

			var allMessages []redis.XMessage
			for _, stream := range messages {
				allMessages = append(allMessages, stream.Messages...)
			}

			if len(allMessages) > 0 {
				select {
				case batchChan <- allMessages:
					logger.Debug("Sent batch to workers", zap.Int("batch_size", len(allMessages)))
				case <-c.ctx.Done():
					return
				}
			}
		}
	}
}

func (c *AbstractConsumer) batchWorker(workerID int, batchChan <-chan []redis.XMessage) {
	defer c.wg.Done()
	logger.Debug("Worker started",
		zap.Int("worker_id", workerID),
		zap.String("stream", c.config.StreamKey))

	for {
		select {
		case <-c.ctx.Done():
			logger.Debug("Worker stopping",
				zap.Int("worker_id", workerID),
				zap.String("stream", c.config.StreamKey))
			return
		case batch, ok := <-batchChan:
			if !ok {
				logger.Debug("Batch channel closed", zap.Int("worker_id", workerID))
				return
			}
			logger.Debug("Worker processing batch",
				zap.Int("worker_id", workerID),
				zap.Int("batch_size", len(batch)))
			c.processBatch(batch)
		}
	}
}

func (c *AbstractConsumer) processBatch(messages []redis.XMessage) {
	if len(messages) == 0 {
		return
	}

	err := c.processor.ProcessBatch(c.ctx, messages)
	if err != nil {
		logger.Error("Error processing batch", logger.WithError(err))
	}

	for _, msg := range messages {
		c.acknowledgeMessage(msg.ID)
	}
}

func (c *AbstractConsumer) acknowledgeMessage(messageID string) {
	if err := c.client.XAck(
		c.ctx,
		c.config.StreamKey,
		c.config.Group,
		messageID,
	).Err(); err != nil {
		logger.Error("Error acknowledging message",
			zap.String("message_id", messageID),
			logger.WithError(err))
	}
}
