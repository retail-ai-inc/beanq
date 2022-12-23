package client

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"beanq/driver"
	"beanq/internal/json"
	"beanq/internal/stringx"
	"beanq/internal/timex"
	"beanq/server"
	"beanq/task"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

var (
	once   sync.Once
	client *redis.Client
)

type BeanqRedis struct {
	client *redis.Client
	wg     *sync.WaitGroup
	stop   chan struct{} // goroutines stop
	ch     chan redis.XStream
	done   chan struct{} // task has done
	err    chan error

	broker                   string
	keepJobInQueue           time.Duration
	keepFailedJobsInHistory  time.Duration
	keepSuccessJobsInHistory time.Duration

	minWorkers  int
	jobMaxRetry int
	prefix      string
}

func NewRedis(options task.Options) *BeanqRedis {
	ctx := context.Background()
	once.Do(func() {
		client = redis.NewClient(options.RedisOptions)
	})
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("RedisError:%+v \n", err)
	}
	return &BeanqRedis{
		client:                   client,
		wg:                       &sync.WaitGroup{},
		stop:                     make(chan struct{}),
		ch:                       make(chan redis.XStream),
		done:                     make(chan struct{}),
		err:                      make(chan error),
		minWorkers:               options.MinWorkers,
		jobMaxRetry:              options.JobMaxRetry,
		prefix:                   options.Prefix,
		keepJobInQueue:           options.KeepJobInQueue,
		keepFailedJobsInHistory:  options.KeepFailedJobsInHistory,
		keepSuccessJobsInHistory: options.KeepSuccessJobsInHistory,
	}
}
func (t *BeanqRedis) DelayPublish(ctx context.Context, taskp *task.Task, delayTime time.Time, option ...driver.Option) (*task.Result, error) {
	option = append(option, driver.ExecuteTime(delayTime))
	return t.Publish(ctx, taskp, option...)
}

func (t *BeanqRedis) Publish(ctx context.Context, taskp *task.Task, option ...driver.Option) (*task.Result, error) {

	opt, err := driver.ComposeOptions(option...)
	if err != nil {
		return nil, err
	}
	// system generation id
	id := "*"

	values := t.args(opt.Queue, taskp.Name, taskp.Payload, opt.Retry, opt.MaxLen, opt.ExecuteTime)

	strcmd := t.client.XAdd(ctx, &redis.XAddArgs{
		Stream:     opt.Queue,
		NoMkStream: false,
		MaxLen:     opt.MaxLen,
		MinID:      "",
		Approx:     false,
		// Limit:      0,
		ID:     id,
		Values: values,
	})

	if err := strcmd.Err(); err != nil {
		return nil, err
	}
	return &task.Result{Args: strcmd.Args(), Id: strcmd.Val()}, nil
}
func (t *BeanqRedis) Start(server *server.Server) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumers := server.Consumers()
	workers := make(chan struct{}, t.minWorkers)

	for _, v := range consumers {

		// if has bound a group,then continue
		result, err := t.client.XInfoGroups(ctx, v.Queue).Result()
		if err != nil && err.Error() != "ERR no such key" {
			fmt.Printf("InfoGroupErr:%+v \n", err)
		}

		if len(result) < 1 {
			if err := t.createGroup(ctx, v.Queue, v.Group); err != nil {
				fmt.Printf("CreateGroupErr:%+v \n", err)
				continue
			}
		}

		workers <- struct{}{}
		go t.work(ctx, v, server, workers)
	}
	// https://redis.io/commands/xclaim/
	// monitor other stream pending
	// go t.claim(consumers)
	// consumer schedule jobs
	go t.delayConsumer(ctx, consumers)
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
func (t *BeanqRedis) StartUI() error {
	return nil
}

/*
* work
*  @Description:
*  @receiver t
* @param handler
* @param server
* @param workers
 */
func (t *BeanqRedis) work(ctx context.Context, handler *server.ConsumerHandler, server *server.Server, workers chan struct{}) {
	defer close(t.done)

	ch, err := t.readGroups(ctx, handler.Queue, handler.Group, server.Count)
	if err != nil {
		t.err <- err
		return
	}
	t.consumerMsgs(ctx, handler.ConsumerFun, handler.Group, ch)
	<-workers
}

/*
  - delayConsumer
  - @Description:
    now testing,need to optimize
  - @receiver t
*/
func (t *BeanqRedis) delayConsumer(ctx context.Context, consumers []*server.ConsumerHandler) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	jn := json.Json

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			for _, consumer := range consumers {
				queueName := consumer.Queue + "-list"
				result, err := t.client.LRange(ctx, queueName, 0, 10).Result()
				if err != nil {
					fmt.Printf("LRangeError:%s \n", err.Error())
					t.err <- fmt.Errorf("LRangeErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					continue
				}
				if len(result) <= 0 {
					continue
				}
				// those codes need to improve
				var taskV task.Task
				for _, s := range result {
					if err := jn.Unmarshal([]byte(s), &taskV); err != nil {
						fmt.Printf("UnmarshalError:%s \n", err.Error())
						t.err <- fmt.Errorf("UnmarshalErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						continue
					}

					if err := t.client.LRem(ctx, queueName, 1, s).Err(); err != nil {
						fmt.Printf("LRemErr:%s \n", err.Error())
						t.err <- fmt.Errorf("LRemErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						continue
					}

					if taskV.ExecuteTime.Before(time.Now()) {

						_, err := t.Publish(ctx, &taskV, driver.Queue(consumer.Queue))
						if err != nil {
							fmt.Printf("PublishError:%s \n", err.Error())
							t.err <- fmt.Errorf("PublishErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						}
						continue
					}
					if err := t.client.RPush(ctx, queueName, s).Err(); err != nil {
						fmt.Printf("RPushError:%s \n", err.Error())
						t.err <- fmt.Errorf("RPushErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						continue
					}
				}
			}
		}
	}
}

/*
  - claim
  - @Description:
    need test
    this function can't work,developing
  - @receiver t
*/
func (t *BeanqRedis) claim(ctx context.Context, consumers []*server.ConsumerHandler) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			start := "-"
			end := "+"

			for _, consumer := range consumers {

				res, err := t.client.XPendingExt(ctx, &redis.XPendingExtArgs{
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

					if v.Idle.Seconds() > 10 {

						claims, err := t.client.XClaim(ctx, &redis.XClaimArgs{
							Stream: consumer.Queue,
							Group:  consumer.Group,
							// Consumer: consumer.queue,
							MinIdle:  10 * time.Second,
							Messages: []string{v.ID},
						}).Result()
						if err != nil && err != redis.Nil {
							t.err <- fmt.Errorf("XClaimErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
							continue
						}
						if err := t.client.XAck(ctx, consumer.Queue, consumer.Group, v.ID).Err(); err != nil {
							t.err <- fmt.Errorf("XAckErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
							continue
						}
						fmt.Printf("claim:%+v \n", claims)

						t.ch <- redis.XStream{
							Stream:   consumer.Queue,
							Messages: claims,
						}
					}
				}
			}
		}
	}
}
func (t *BeanqRedis) readGroups(ctx context.Context, queue, group string, count int64) (<-chan redis.XStream, error) {
	consumer := uuid.New().String()
	ch := make(chan redis.XStream)
	go func() {

		for {
			select {
			case <-ctx.Done():
				return
			default:
				streams, err := t.client.XReadGroup(ctx, &redis.XReadGroupArgs{
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
					ch <- v
				}
			}
		}
	}()
	return ch, nil
}

func (t *BeanqRedis) consumerMsgs(ctx context.Context, f task.DoConsumer, group string, ch <-chan redis.XStream) {
	info := task.SuccessInfo
	result := &task.ConsumerResult{
		Level:   task.InfoLevel,
		Info:    info,
		RunTime: "",
	}
	var now time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			taskp := &task.Task{
				Name: msg.Stream,
			}
			for _, vm := range msg.Messages {

				t.parseMapToTask(taskp, vm)
				now = time.Now()
				if taskp.ExecuteTime.After(now) {
					// format data
					maps := t.args(msg.Stream, taskp.Name, taskp.Payload, taskp.Retry, taskp.MaxLen, taskp.ExecuteTime)
					data, err := json.Json.MarshalToString(maps)
					if err != nil {

					}
					if err := t.client.LPush(ctx, msg.Stream+"-list", data).Err(); err != nil {

					}
					continue
				} else {
					err := t.retry(func() error {
						return f(taskp, t.client)
					}, task.DefaultOptions.RetryTime)

					if err != nil {
						info = task.FailedInfo
						result.Level = task.ErrLevel
						result.Info = task.FlagInfo(err.Error())
					}

					sub := time.Now().Sub(now)

					result.Payload = taskp.Payload
					result.AddTime = time.Now().Format(timex.DateTime)
					result.RunTime = sub.String()
					result.Queue = msg.Stream
					result.Group = group

					b, err := json.Marshal(result)
					if err != nil {
						t.err <- fmt.Errorf("JsonMarshalErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						continue
					}
					if err := t.client.LPush(ctx, string(info), b).Err(); err != nil {
						t.err <- fmt.Errorf("LPushErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
						continue
					}
				}
				// ack
				if err := t.client.XAck(ctx, msg.Stream, group, vm.ID).Err(); err != nil {
					t.err <- fmt.Errorf("XACKErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					fmt.Printf("ACK Error:%s \n", err.Error())
					continue
				}
				if err := t.client.XDel(ctx, msg.Stream, vm.ID).Err(); err != nil {
					t.err <- fmt.Errorf("XdelErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					continue
				}
			}
		}
	}
}
func (t *BeanqRedis) retry(f func() error, delayTime time.Duration) error {
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

/*
  - createGroup
  - @Description:
    if group not exist,then create it
  - @receiver t
  - @param queue
  - @param group
  - @return error
*/
func (t *BeanqRedis) createGroup(ctx context.Context, queue, group string) error {

	cmd := t.client.XGroupCreateMkStream(ctx, queue, group, "0")
	if cmd.Err() != nil && cmd.Err().Error() != "BUSYGROUP Consumer Group name already exists" {
		return cmd.Err()
	}
	return nil

}
func (t *BeanqRedis) args(queue, name string, payload []byte, retry int, maxLen int64, executeTime time.Time) map[string]any {
	values := make(map[string]any)
	values["queue"] = queue
	values["name"] = name
	values["payload"] = payload
	values["addtime"] = time.Now().Format(timex.DateTime)
	values["retry"] = retry
	values["maxLen"] = maxLen

	if !executeTime.IsZero() {
		values["executeTime"] = executeTime
	}
	return values
}
func (t *BeanqRedis) parseMapToTask(task2 *task.Task, msg redis.XMessage) {

	task2.Id = msg.ID
	if queue, ok := msg.Values["queue"]; ok {
		if v, okv := queue.(string); okv {
			task2.Queue = v
		}
	}
	if maxLen, ok := msg.Values["maxLen"]; ok {
		if v, okv := maxLen.(string); okv {
			task2.MaxLen = cast.ToInt64(v)
		}
	}
	if retry, ok := msg.Values["retry"]; ok {
		if v, okv := retry.(string); okv {
			task2.Retry = cast.ToInt(v)
		}
	}
	if payload, ok := msg.Values["payload"]; ok {
		if payloadV, okp := payload.(string); okp {
			task2.Payload = stringx.StringToByte(payloadV)
		}
	}
	if addtime, ok := msg.Values["addtime"]; ok {
		if addtimeV, okt := addtime.(string); okt {
			task2.AddTime = addtimeV
		}
	}
	if executeT, ok := msg.Values["executeTime"]; ok {
		if executeTm, okt := executeT.(string); okt {
			task2.ExecuteTime = cast.ToTime(executeTm)
		}
	}
}

/*
  - GetErrors
  - @Description:
    can't get error msg,need to optimize
  - @receiver t
  - @return err
*/
func (t *BeanqRedis) GetErrors() (err error) {
	for {
		select {
		case errc := <-t.err:
			err = errc
			return
		}
	}
}

/*
  - Close
  - @Description:
    close redis client
  - @receiver t
  - @return error
*/
func (t *BeanqRedis) Close() error {
	t.stop <- struct{}{}
	return t.client.Close()
}
