package bredis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/spf13/cast"
)

const (
	deadLetterRetryField          = "deadLetterRetry"
	maxDeadLetterRetry            = 3
	defaultDeadLetterMaxLen int64 = 200000
)

type deadLetterProcessor struct {
	client        redis.UniversalClient
	captureConfig *capture.Config
	channel       string
	topic         string
	streamKey     string
	lockKey       string
	prefix        string
	idle          time.Duration
	wait          replicationWait
}

type deadLetterMessage struct {
	values map[string]any
}

func (t *queueBase) DeadLetterStream(ctx context.Context, channel, topic, streamKey, lockKey string) {
	processor := deadLetterProcessor{
		client:        t.client,
		captureConfig: t.captureConfig,
		channel:       channel,
		topic:         topic,
		streamKey:     streamKey,
		lockKey:       lockKey,
		prefix:        t.prefix,
		idle:          t.deadLetterIdle,
		wait:          t.wait,
	}
	processor.run(ctx)
}

func (p *deadLetterProcessor) run(ctx context.Context) {
	ticker := time.NewTicker(DefaultBlockDuration())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.scan(ctx)
		}
	}
}

func (p *deadLetterProcessor) scan(ctx context.Context) {
	if p.locked(ctx) {
		return
	}
	defer p.releaseLock(ctx)

	pending, ok := p.oldestExpiredPending(ctx)
	if !ok {
		return
	}
	messages := p.client.XRange(ctx, p.streamKey, pending.ID, pending.ID).Val()
	if len(messages) == 0 {
		return
	}
	if err := p.move(ctx, pending.ID, deadLetterMessage{values: messages[0].Values}); err != nil {
		p.report(err)
	}
}

func (p *deadLetterProcessor) locked(ctx context.Context) bool {
	return cast.ToInt64(AddLogicLockScript.Run(ctx, p.client, []string{p.lockKey}).Val()) == 1
}

func (p *deadLetterProcessor) oldestExpiredPending(ctx context.Context) (redis.XPendingExt, bool) {
	pendings := p.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: p.streamKey, Group: p.channel, Start: "-", End: "+", Count: 1,
	}).Val()
	if len(pendings) == 0 || pendings[0].Idle <= p.idle {
		return redis.XPendingExt{}, false
	}
	return pendings[0], true
}

func (p *deadLetterProcessor) move(ctx context.Context, pendingID string, message deadLetterMessage) error {
	logicKey := tool.MakeLogicKeyForID(p.prefix, cast.ToString(message.values["id"]))
	args := message.xAddArgs(p.streamKey, logicKey)
	if p.wait.enabled() {
		if _, err := p.wait.execute(ctx, p.client, args.Stream, func(pipe redis.Pipeliner) redis.Cmder {
			return pipe.XAdd(ctx, args)
		}); err != nil {
			return err
		}
	} else if err := p.client.XAdd(ctx, args).Err(); err != nil {
		return err
	}
	if p.wait.enabled() {
		return ackAndDelete(ctx, p.client, p.wait, p.streamKey, p.channel, pendingID)
	}
	_, err := p.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.XAck(ctx, p.streamKey, p.channel, pendingID)
		pipe.XDel(ctx, p.streamKey, pendingID)
		return nil
	})
	return err
}

func (p *deadLetterProcessor) releaseLock(ctx context.Context) {
	if err := p.client.Unlink(ctx, p.lockKey).Err(); err != nil && !errors.Is(err, context.Canceled) {
		p.report(err)
	}
}

func (p *deadLetterProcessor) report(err error) {
	capture.Dlq.When(p.captureConfig).If(&capture.Channel{Channel: p.channel, Topic: []string{p.topic}}).Then(err)
	logger.New().Error(err)
}

func (m deadLetterMessage) xAddArgs(streamKey, logicKey string) *redis.XAddArgs {
	retry := m.retryCount()
	m.values[deadLetterRetryField] = retry
	if retry >= maxDeadLetterRetry {
		m.values["logType"] = bstatus.Dlq
		m.values["status"] = bstatus.StatusFailed
		return &redis.XAddArgs{
			Stream: logicKey, MaxLen: defaultLogicLogMaxLen, Approx: true, ID: "*", Values: m.values,
		}
	}
	m.incrementRetry()
	delete(m.values, "logType")
	return NewZAddArgs(streamKey, "", "*", m.maxLen(), 0, m.values)
}

func (m deadLetterMessage) retryCount() int {
	return cast.ToInt(m.values[deadLetterRetryField])
}

func (m deadLetterMessage) incrementRetry() int {
	retry := m.retryCount() + 1
	m.values[deadLetterRetryField] = retry
	return retry
}

func (m deadLetterMessage) maxLen() int64 {
	maxLen := cast.ToInt64(m.values["maxLen"])
	if maxLen <= 0 {
		return defaultDeadLetterMaxLen
	}
	return maxLen
}
