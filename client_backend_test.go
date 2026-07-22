package beanq

import (
	"context"
	"errors"
	"testing"
	"time"

	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
)

type fakeQueueBackend struct {
	published        chan map[string]any
	started          chan btype.MoodType
	shutdown         <-chan struct{}
	stopped          chan struct{}
	ack              map[string]string
	sequenceAck      map[string]string
	sequenceOrderKey string
	err              error
}

func (f *fakeQueueBackend) Enqueue(_ context.Context, data map[string]any) error {
	if f.published != nil {
		f.published <- data
	}
	return f.err
}

func (f *fakeQueueBackend) Consume(ctx context.Context, mood btype.MoodType, _, _ string, _ public.CallbackWithRetry) error {
	if f.err != nil {
		return f.err
	}
	if f.started != nil {
		f.started <- mood
	}
	if f.started != nil || f.shutdown != nil {
		<-ctx.Done()
		if f.shutdown != nil {
			<-f.shutdown
		}
		if f.stopped != nil {
			close(f.stopped)
		}
	}
	return nil
}

func (f *fakeQueueBackend) WaitingAck(_ context.Context, _, _, _ string) (map[string]string, error) {
	return f.ack, f.err
}

func (f *fakeQueueBackend) WaitingSequenceAck(_ context.Context, _, _, orderKey, _ string) (map[string]string, error) {
	f.sequenceOrderKey = orderKey
	if f.sequenceAck != nil {
		return f.sequenceAck, f.err
	}
	return f.ack, f.err
}

func TestClientPublishUsesBackendNeutralPublisher(t *testing.T) {
	published := make(chan map[string]any, 1)
	b := &fakeQueueBackend{published: published}
	c := &Client{broker: b, Channel: "default-channel", Topic: "default-topic", MaxLen: 10}

	if err := c.BQ().WithContext(context.Background()).PublishAtTime("channel", "topic", []byte("payload"), time.Now()); err != nil {
		t.Fatalf("PublishAtTime: %v", err)
	}
	message := <-published
	if message["moodType"] != btype.DELAY || message["channel"] != "channel" || message["topic"] != "topic" {
		t.Fatalf("unexpected published message: %#v", message)
	}
}

func TestClientStartsBackendNeutralConsumer(t *testing.T) {
	started := make(chan btype.MoodType, 1)
	b := &fakeQueueBackend{started: started}
	c := &Client{broker: b, consumers: &consumerRegistry{handlers: []*Handler{{moodType: btype.DELAY, do: public.NewCallbackWithRetry(nil, nil)}}}}
	ctx, cancel := context.WithCancel(context.Background())
	c.startHandlers(ctx)
	select {
	case mood := <-started:
		if mood != btype.DELAY {
			t.Fatalf("consumer mood = %q, want delay", mood)
		}
	case <-time.After(time.Second):
		t.Fatal("consumer did not start")
	}
	cancel()
}

func TestCommandClonesRegisterConsumersOnRootClient(t *testing.T) {
	started := make(chan btype.MoodType, 5)
	b := &fakeQueueBackend{started: started}
	c := &Client{broker: b, consumers: &consumerRegistry{}}
	handle := DefaultHandle{DoHandle: func(context.Context, *Message) error { return nil }}

	registrations := []struct {
		mood     btype.MoodType
		register func() error
	}{
		{btype.NORMAL, func() error { _, err := c.BQ().Subscribe("channel", "normal", handle); return err }},
		{btype.DELAY, func() error { _, err := c.BQ().SubscribeToDelay("channel", "delay", handle); return err }},
		{btype.SEQUENCE_QUEUE, func() error { _, err := c.BQ().SubscribeSequence("channel", "sequence-queue", handle); return err }},
	}
	for _, registration := range registrations {
		if err := registration.register(); err != nil {
			t.Fatalf("register %q consumer: %v", registration.mood, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.startHandlers(ctx)

	got := make(map[btype.MoodType]int, len(registrations))
	for range registrations {
		select {
		case mood := <-started:
			got[mood]++
		case <-time.After(time.Second):
			t.Fatal("registered consumer did not start")
		}
	}
	for _, registration := range registrations {
		if got[registration.mood] != 1 {
			t.Fatalf("consumer mood %q started %d times, want 1", registration.mood, got[registration.mood])
		}
	}

	c.startHandlers(ctx)
	select {
	case mood := <-started:
		t.Fatalf("consumer mood %q started again after registry was drained", mood)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestClientWaitReturnsWhenParentContextIsCanceled(t *testing.T) {
	c := &Client{broker: &fakeQueueBackend{}, consumers: &consumerRegistry{}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		c.Wait(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after parent context cancellation")
	}
}

func TestClientWaitAllowsConsumerGracefulShutdown(t *testing.T) {
	started := make(chan btype.MoodType, 1)
	releaseShutdown := make(chan struct{})
	stopped := make(chan struct{})
	b := &fakeQueueBackend{started: started, shutdown: releaseShutdown, stopped: stopped}
	c := &Client{broker: b, consumers: &consumerRegistry{}}
	handle := DefaultHandle{DoHandle: func(context.Context, *Message) error { return nil }}
	if _, err := c.BQ().Subscribe("channel", "topic", handle); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	waitDone := make(chan struct{})
	go func() {
		c.Wait(ctx)
		close(waitDone)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("consumer did not start")
	}

	cancel()
	select {
	case <-waitDone:
		t.Fatal("Wait returned before consumer completed graceful shutdown")
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseShutdown)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("consumer did not complete shutdown")
	}
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after consumer shutdown")
	}
}

func TestBrokerConsumerResolverErrorIsReturned(t *testing.T) {
	want := errors.New("consumer unavailable")
	b := &fakeQueueBackend{err: want}
	err := b.Consume(context.Background(), btype.NORMAL, "channel", "topic", public.NewCallbackWithRetry(nil, nil))
	if !errors.Is(err, want) {
		t.Fatalf("Consumer error = %v, want %v", err, want)
	}
}

func TestSequenceWaitingAckUsesBackendNeutralContract(t *testing.T) {
	b := &fakeQueueBackend{sequenceAck: map[string]string{"id": "message-id"}}
	client := &Client{broker: b}

	message, err := (&SequenceCmd{client: client, ctx: context.Background(), id: "message-id", orderKey: "order-1"}).WaitingAck()
	if err != nil {
		t.Fatalf("WaitingAck: %v", err)
	}
	if message.Id != "message-id" {
		t.Fatalf("message id = %q, want message-id", message.Id)
	}
	if b.sequenceOrderKey != "order-1" {
		t.Fatalf("sequence ack orderKey = %q, want order-1", b.sequenceOrderKey)
	}
}
