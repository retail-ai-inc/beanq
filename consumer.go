package beanq

import (
	"context"

	opt "beanq/internal/options"
)

type Consumer struct {
	broker Broker
	opts   *opt.Options
}

var _ BeanqSub = new(Consumer)

func NewConsumer(broker Broker, options *opt.Options) *Consumer {
	opts := opt.DefaultOptions
	if options != nil {
		if options.KeepJobInQueue != 0 {
			opts.KeepJobInQueue = options.KeepJobInQueue
		}

		if options.KeepFailedJobsInHistory != 0 {
			opts.KeepFailedJobsInHistory = options.KeepFailedJobsInHistory
		}

		if options.KeepSuccessJobsInHistory != 0 {
			opts.KeepSuccessJobsInHistory = options.KeepSuccessJobsInHistory
		}

		if options.MinWorkers != 0 {
			opts.MinWorkers = options.MinWorkers
		}

		if options.JobMaxRetry != 0 {
			opts.JobMaxRetry = options.JobMaxRetry
		}

		if options.Prefix != "" {
			opts.Prefix = options.Prefix
		}

		if options.DefaultQueueName != "" {
			opts.DefaultQueueName = options.DefaultDelayQueueName
		}

		if options.DefaultGroup != "" {
			opts.DefaultGroup = options.DefaultGroup
		}

		if options.DefaultMaxLen != 0 {
			opts.DefaultMaxLen = options.DefaultMaxLen
		}

		if options.DefaultDelayQueueName != "" {
			opts.DefaultDelayQueueName = options.DefaultDelayQueueName
		}

		if options.DefaultDelayGroup != "" {
			opts.DefaultDelayGroup = options.DefaultDelayGroup
		}

		if options.RetryTime != 0 {
			opts.RetryTime = options.RetryTime
		}
	}

	return &Consumer{broker: broker, opts: opts}
}

func (t *Consumer) StartConsumerWithContext(ctx context.Context, srv *Server) {

	t.broker.start(ctx, srv)

}

func (t *Consumer) StartConsumer(srv *Server) {
	ctx := context.Background()
	t.StartConsumerWithContext(ctx, srv)
}
func (t *Consumer) StartUI() error {
	return nil
}
