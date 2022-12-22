package beanq

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"beanq/helper/json"
	"beanq/helper/stringx"
	"beanq/helper/timex"
	"beanq/internal/base"
	"beanq/internal/driver"
	opt "beanq/internal/options"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

type RedisBroker struct {
	client     *redis.Client
	ctx        context.Context
	done, stop chan struct{}
	minWorkers int
	err        chan error
}

var _ Broker = new(RedisBroker)

func NewRedisBroker(options2 *redis.Options) *RedisBroker {
	return &RedisBroker{
		client:     driver.NewRdb(options2),
		ctx:        context.Background(),
		done:       make(chan struct{}),
		stop:       make(chan struct{}),
		minWorkers: 10,
		err:        make(chan error),
	}
}

func (t *RedisBroker) Enqueue(ctx context.Context, values map[string]any, opts opt.Option) (*opt.Result, error) {
	id := "*"
	strcmd := t.client.XAdd(ctx, &redis.XAddArgs{
		Stream:     opts.Queue,
		NoMkStream: false,
		MaxLen:     opts.MaxLen,
		MinID:      "",
		Approx:     false,
		// Limit:      0,
		ID:     id,
		Values: values,
	})
	if err := strcmd.Err(); err != nil {
		return nil, err
	}
	return &opt.Result{Args: strcmd.Args(), Id: strcmd.Val()}, nil
}
func (t *RedisBroker) Start(ctx context.Context, server *Server) {
	consumers := server.Consumers()
	workers := make(chan struct{}, t.minWorkers)
	t.ctx = ctx

	for _, v := range consumers {

		// if has bound a group,then continue
		result, err := t.client.XInfoGroups(t.ctx, v.Queue).Result()
		if err != nil && err.Error() != "ERR no such key" {
			fmt.Printf("InfoGroupErr:%+v \n", err)
		}
		if len(result) < 1 {
			if err := t.createGroup(v.Queue, v.Group); err != nil {
				fmt.Printf("CreateGroupErr:%+v \n", err)
				continue
			}
		}

		workers <- struct{}{}
		go t.work(v, server, workers)
	}
	//https://redis.io/commands/xclaim/
	//monitor other stream pending
	go t.claim(consumers)
	//consumer schedule jobs

	go t.delayConsumer(consumers)
	// catch errors
	<-t.done
}
func (t *RedisBroker) work(handler *ConsumerHandler, server *Server, workers chan struct{}) {
	defer close(t.done)
	ch, err := t.readGroups(handler.Queue, handler.Group, server.Count)
	if err != nil {
		t.err <- err
		return
	}
	t.consumerMsgs(handler.ConsumerFun, handler.Group, ch)
	<-workers
}
func (t *RedisBroker) createGroup(queue, group string) error {
	cmd := t.client.XGroupCreateMkStream(t.ctx, queue, group, "0")
	if cmd.Err() != nil && cmd.Err().Error() != "BUSYGROUP Consumer Group name already exists" {
		return cmd.Err()
	}
	return nil
}
func (t *RedisBroker) readGroups(queue, group string, count int64) (<-chan *redis.XStream, error) {
	consumer := uuid.New().String()
	ch := make(chan *redis.XStream)
	go func() {

		for {
			select {
			case <-t.stop:
				return
			default:
				streams, err := t.client.XReadGroup(t.ctx, &redis.XReadGroupArgs{
					Group:    group,
					Streams:  []string{queue, ">"},
					Consumer: consumer,
					Count:    count,
					Block:    0,
				}).Result()

				if err != nil {
					t.err <- fmt.Errorf("XReadGroupErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					continue
				}
				if len(streams) <= 0 {
					continue
				}

				for _, v := range streams {
					ch <- &v
				}
			}
		}
	}()
	return ch, nil
}
func (t *RedisBroker) claim(consumers []*ConsumerHandler) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			start := "-"
			end := "+"

			for _, consumer := range consumers {

				res, err := t.client.XPendingExt(t.ctx, &redis.XPendingExtArgs{
					Stream: consumer.Queue,
					Group:  consumer.Group,
					Start:  start,
					End:    end,
					// Count:  10,
				}).Result()
				if err != nil && err != redis.Nil {
					t.err <- fmt.Errorf("XPendingErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					break
				}
				for _, v := range res {

					if v.Idle.Seconds() > 60 {
						uuid := uuid.New().String()
						claims, err := t.client.XClaim(t.ctx, &redis.XClaimArgs{

							Stream:   consumer.Queue,
							Group:    consumer.Group,
							Consumer: uuid,
							MinIdle:  60 * time.Second,

							Messages: []string{v.ID},
						}).Result()
						if err != nil && err != redis.Nil {
							t.err <- fmt.Errorf("XClaimErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
							continue
						}
						ch := make(chan *redis.XStream, 1)
						ch <- &redis.XStream{
							Stream:   consumer.Queue,
							Messages: claims,
						}
						t.consumerMsgs(consumer.ConsumerFun, consumer.Group, ch)
						close(ch)
						fmt.Printf("claim:%+v \n", claims)
					}
				}
			}
		}
	}
}
func (t *RedisBroker) consumerMsgs(f DoConsumer, group string, ch <-chan *redis.XStream) {
	info := opt.SuccessInfo
	result := &opt.ConsumerResult{
		Level:   opt.InfoLevel,
		Info:    info,
		RunTime: "",
	}
	var now time.Time

	for {
		select {
		case <-t.stop:
			return
		case msg := <-ch:

			stream := msg.Stream
			for _, vm := range msg.Messages {

				taskp := t.parseMapToTask(vm, stream)
				now = time.Now()

				if taskp.ExecuteTime().After(now) {
					//format data
					maps := base.ParseArgs(msg.Stream, taskp.Name(), taskp.Payload(), taskp.Retry(), taskp.MaxLen(), taskp.ExecuteTime())

					data, err := json.Json.MarshalToString(maps)
					if err != nil {

					}
					if err := t.client.LPush(t.ctx, msg.Stream+"-list", data).Err(); err != nil {

					}
					continue
				} else {
					err := t.retry(func() error {
						return f(taskp, t.client)
					}, opt.DefaultOptions.RetryTime)

					if err != nil {
						info = opt.FailedInfo
						result.Level = opt.ErrLevel
						result.Info = opt.FlagInfo(err.Error())
					}

					sub := time.Now().Sub(now)

					result.Payload = taskp.Payload()
					result.AddTime = time.Now().Format(timex.DateTime)
					result.RunTime = sub.String()
					result.Queue = msg.Stream
					result.Group = group

					b, err := json.Marshal(result)
					if err != nil {
						t.err <- fmt.Errorf("JsonMarshalErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						continue
					}
					if err := t.client.LPush(t.ctx, string(info), b).Err(); err != nil {
						t.err <- fmt.Errorf("LPushErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						continue
					}
				}
				// ack
				if err := t.client.XAck(t.ctx, msg.Stream, group, vm.ID).Err(); err != nil {
					t.err <- fmt.Errorf("XACKErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					fmt.Printf("ACK Error:%s \n", err.Error())
					continue
				}
				if err := t.client.XDel(t.ctx, msg.Stream, vm.ID).Err(); err != nil {
					t.err <- fmt.Errorf("XdelErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					continue
				}
			}
		}
	}
}
func (t *RedisBroker) delayConsumer(consumers []*ConsumerHandler) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	jn := json.Json

	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:

			for _, consumer := range consumers {

				queueName := consumer.Queue + "-list"
				result, err := t.client.LRange(t.ctx, queueName, 0, 10).Result()
				if err != nil {
					fmt.Printf("LRangeError:%s \n", err.Error())
					t.err <- fmt.Errorf("LRangeErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					continue
				}
				if len(result) <= 0 {
					continue
				}
				//those codes need to improve
				for _, s := range result {

					if err := t.client.LRem(t.ctx, queueName, 1, s).Err(); err != nil {
						fmt.Printf("LRemErr:%s \n", err.Error())
						t.err <- fmt.Errorf("LRemErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						continue
					}
					bt := []byte(s)
					executeTime := cast.ToTime(jn.Get(bt, "executeTime").ToString())

					if executeTime.Before(time.Now()) {

						name := jn.Get(bt, "name").ToString()
						payload := jn.Get(bt, "payload").ToString()
						queue := jn.Get(bt, "queue").ToString()
						retry := jn.Get(bt, "retry").ToInt()
						maxLen := jn.Get(bt, "maxLen").ToInt64()

						values := base.ParseArgs(queue, name, payload, retry, maxLen, executeTime)
						opts := opt.Option{
							Queue:  queue,
							MaxLen: maxLen,
						}
						_, err = t.Enqueue(t.ctx, values, opts)
						if err != nil {
							fmt.Printf("PublishError:%s \n", err.Error())
							t.err <- fmt.Errorf("PublishErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						}
						continue
					}
					if err := t.client.RPush(t.ctx, queueName, s).Err(); err != nil {
						fmt.Printf("RPushError:%s \n", err.Error())
						t.err <- fmt.Errorf("RPushErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						continue
					}
				}
			}
		}
	}
}
func (t *RedisBroker) retry(f func() error, delayTime time.Duration) error {
	retryFlag := make(chan error)
	stopRetry := make(chan bool, 1)

	go func(duration time.Duration, errChan chan error, stop chan bool) {
		index := 1
		count := 3

		for {
			go time.AfterFunc(duration, func() {
				errChan <- f()
			})

			err := <-errChan
			if err == nil {
				stop <- true
				close(errChan)
				break
			}
			if index == count {
				stop <- true
				errChan <- err
				break
			}
			index++
		}
	}(delayTime, retryFlag, stopRetry)

	var err error
	select {
	case <-stopRetry:
		for v := range retryFlag {
			err = v
			if v != nil {
				err = v
				break
			}
		}
	}
	close(stopRetry)
	if err != nil {
		close(retryFlag)
		return err
	}
	return nil
}
func (t *RedisBroker) Close() error {
	return t.client.Close()
}
func (t *RedisBroker) parseMapToTask(msg redis.XMessage, stream string) *Task {

	id := msg.ID
	var queueStr string
	var maxLenV int64
	var retryV int
	var payloadB []byte
	var addTimeStr string
	var executeTime time.Time

	if queue, ok := msg.Values["queue"]; ok {
		if v, okv := queue.(string); okv {
			queueStr = v
		}
	}
	if maxLen, ok := msg.Values["maxLen"]; ok {
		if v, okv := maxLen.(string); okv {
			maxLenV = cast.ToInt64(v)
		}
	}
	if retry, ok := msg.Values["retry"]; ok {
		if v, okv := retry.(string); okv {
			retryV = cast.ToInt(v)
		}
	}
	if payload, ok := msg.Values["payload"]; ok {
		if payloadV, okp := payload.(string); okp {
			payloadB = stringx.StringToByte(payloadV)
		}
	}
	if addtime, ok := msg.Values["addtime"]; ok {
		if addtimeV, okt := addtime.(string); okt {
			addTimeStr = addtimeV
		}
	}
	if executeT, ok := msg.Values["executeTime"]; ok {
		if executeTm, okt := executeT.(string); okt {
			executeTime = cast.ToTime(executeTm)
		}
	}
	return NewTask(
		SetId(id),
		SetName(stream),
		SetAddTime(addTimeStr),
		SetExecuteTime(executeTime),
		SetMaxLen(maxLenV),
		SetPayLoad(payloadB),
		SetQueue(queueStr),
		SetRetry(retryV))
}
