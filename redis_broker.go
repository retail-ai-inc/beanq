package beanq

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"beanq/helper/json"
	"beanq/helper/stringx"
	"beanq/helper/timex"
	"beanq/internal/base"
	"beanq/internal/driver"
	opt "beanq/internal/options"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type RedisBroker struct {
	client      *redis.Client
	ctx         context.Context
	done, stop  chan struct{}
	minWorkers  int
	err         chan error
	healthCheck healthCheckI
	scheduleJob scheduleJobI
}

var _ Broker = new(RedisBroker)

func NewRedisBroker(config BeanqConfig) *RedisBroker {
	client := driver.NewRdb(&redis.Options{
		Addr:         config.Queue.Redis.Host + ":" + config.Queue.Redis.Port,
		Password:     config.Queue.Redis.Password,
		DB:           config.Queue.Redis.Database,
		MaxRetries:   config.Queue.Redis.Maxretries,
		PoolSize:     config.Queue.Redis.PoolSize,
		MinIdleConns: config.Queue.Redis.MinIdleConnections,
		DialTimeout:  config.Queue.Redis.DialTimeout,
		ReadTimeout:  config.Queue.Redis.ReadTimeout,
		WriteTimeout: config.Queue.Redis.WriteTimeout,
		PoolTimeout:  config.Queue.Redis.PoolTimeout,
	})

	return &RedisBroker{
		client:      client,
		ctx:         nil,
		done:        make(chan struct{}),
		stop:        make(chan struct{}),
		minWorkers:  10,
		err:         make(chan error, 1),
		healthCheck: newHealthCheck(client),
		scheduleJob: newScheduleJob(client),
	}
}

func (t *RedisBroker) enqueue(ctx context.Context, stream string, task *Task, opts opt.Option) (*opt.Result, error) {

	if stream == "" || task == nil {
		return nil, fmt.Errorf("stream or values can't empty")
	}
	if err := t.scheduleJob.enqueue(ctx, stream, task, opts); err != nil {
		return nil, err
	}
	return nil, nil
}
func (t *RedisBroker) start(ctx context.Context, server *Server) {
	consumers := server.Consumers()

	t.ctx = ctx

	go t.worker(consumers, server)
	// consumer schedule jobs
	go t.scheduleJob.start(ctx, consumers)

	// REFERENCE: https://redis.io/commands/xclaim/
	// monitor other stream pending
	// go t.claim(consumers)

	// check client health
	go t.healthCheckerStart()

	// catch errors
	select {
	case <-t.stop:
		fmt.Println("stop")
		return
	case <-t.done:
		fmt.Println("done")
		return
	}
}

/*
* healthCheckerStart
*  @Description:
*  @receiver t
 */
func (t *RedisBroker) healthCheckerStart() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.ctx.Done():
			t.err <- t.ctx.Err()
			return
		case <-ticker.C:
			if err := t.healthCheck.start(t.ctx); err != nil {
				t.err <- err
				return
			}
		case <-t.stop:
			return
		}
	}
}
func (t *RedisBroker) worker(consumers []*ConsumerHandler, server *Server) {
	workers := make(chan struct{}, t.minWorkers)

	for _, v := range consumers {
		// if has bound a group,then continue
		result, err := t.client.XInfoGroups(t.ctx, base.MakeStreamKey(v.Group, v.Queue)).Result()
		if err != nil && err.Error() != "ERR no such key" {
			t.err <- err
			fmt.Printf("InfoGroupErr:%+v \n", err)
		}

		if len(result) < 1 {
			if err := t.createGroup(v.Queue, v.Group); err != nil {
				fmt.Printf("CreateGroupErr:%+v \n", err)
				t.err <- err
				continue
			}
		}

		workers <- struct{}{}
		go t.work(v, server, workers)
	}
}
func (t *RedisBroker) work(handler *ConsumerHandler, server *Server, workers chan struct{}) {
	defer close(t.done)

	ch, err := t.readGroups(handler.Queue, handler.Group, int64(server.Count))

	if err != nil {
		t.err <- err
		return
	}

	t.consumer(handler.ConsumerFun, handler.Group, ch)
	<-workers
}

func (t *RedisBroker) createGroup(queue, group string) error {
	cmd := t.client.XGroupCreateMkStream(t.ctx, base.MakeStreamKey(group, queue), group, "0")
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
			case <-t.ctx.Done():
				t.err <- t.ctx.Err()
				return
			default:
				streams, err := t.client.XReadGroup(t.ctx, &redis.XReadGroupArgs{
					Group:    group,
					Streams:  []string{base.MakeStreamKey(group, queue), ">"},
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
		case <-t.ctx.Done():
			t.err <- t.ctx.Err()
			return
		case <-ticker.C:
			start := "-"
			end := "+"

			for _, consumer := range consumers {
				res, err := t.client.XPendingExt(t.ctx, &redis.XPendingExtArgs{
					Stream: base.MakeStreamKey(consumer.Group, consumer.Queue),
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

						claims, err := t.client.XClaim(t.ctx, &redis.XClaimArgs{

							Stream:   base.MakeStreamKey(consumer.Group, consumer.Queue),
							Group:    consumer.Group,
							Consumer: consumer.Queue,
							MinIdle:  60 * time.Second,

							Messages: []string{v.ID},
						}).Result()
						if err != nil && err != redis.Nil {
							t.err <- fmt.Errorf("XClaimErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
							continue
						}
						ch := make(chan *redis.XStream, 1)
						ch <- &redis.XStream{
							Stream:   base.MakeStreamKey(consumer.Group, consumer.Queue),
							Messages: claims,
						}
						t.consumer(consumer.ConsumerFun, consumer.Group, ch)
						close(ch)
					}
				}
			}
		}
	}
}

func (t *RedisBroker) consumer(f DoConsumer, group string, ch <-chan *redis.XStream) {
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
		case <-t.ctx.Done():
			t.err <- t.ctx.Err()
			return
		case msg := <-ch:

			stream := msg.Stream
			for _, vm := range msg.Messages {

				task := t.parseMapToTask(vm, stream)
				now = time.Now()

				err := base.Retry(func() error {
					return f(task)
				}, opt.DefaultOptions.RetryTime)
				if err != nil {
					info = opt.FailedInfo
					result.Level = opt.ErrLevel
					result.Info = opt.FlagInfo(err.Error())
				}

				sub := time.Now().Sub(now)

				result.Payload = task.Payload()
				result.AddTime = time.Now().Format(timex.DateTime)
				result.RunTime = sub.String()
				result.Queue = msg.Stream
				result.Group = group

				if err := t.logInToList(result); err != nil {
					t.err <- err
					continue
				}

				// ack
				if err := t.client.XAck(t.ctx, base.MakeStreamKey(group, msg.Stream), group, vm.ID).Err(); err != nil {
					t.err <- fmt.Errorf("XACKErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					fmt.Printf("ACK Error:%s \n", err.Error())
					continue
				}
				if err := t.client.XDel(t.ctx, base.MakeStreamKey(group, msg.Stream), vm.ID).Err(); err != nil {
					t.err <- fmt.Errorf("XdelErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					continue
				}
			}
		}
	}
}

/*
  - logToList
  - @Description:
    push logs to redis list
  - @receiver t
  - @param result
  - @return error
*/
func (t *RedisBroker) logInToList(result *opt.ConsumerResult) error {

	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("JsonMarshalErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
	}
	if err := t.client.LPush(t.ctx, string(result.Info), b).Err(); err != nil {
		return fmt.Errorf("LPushErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
	}
	return nil

}

func (t *RedisBroker) close() error {
	select {
	case <-t.stop:
	default:
		close(t.stop)
	}
	return t.client.Close()
}

/*
  - Error
  - @Description:
    this function can't get errors always,need improve
  - @receiver t
  - @return error
*/
func (t *RedisBroker) Error() error {
	select {
	case err := <-t.err:
		return err
	default:
		return nil
	}
}

func (t *RedisBroker) parseMapToTask(msg redis.XMessage, stream string) *Task {
	payload, id, streamStr, addTime, queue, group, executeTime, retry, maxLen := openTaskMap(BqMessage(msg), stream)
	return &Task{
		Values: values{
			"id":          id,
			"name":        streamStr,
			"queue":       queue,
			"group":       group,
			"maxLen":      maxLen,
			"retry":       retry,
			"payload":     payload,
			"addTime":     addTime,
			"executeTime": executeTime,
		},
		rw: new(sync.RWMutex),
	}
}
