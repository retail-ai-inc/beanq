package beanq

import (
	"context"
	"errors"
	"testing"
)

type cancelTestHandle struct {
	cancelErr error
}

func (h cancelTestHandle) Handle(context.Context, *Message) error { return nil }

func (h cancelTestHandle) Cancel(context.Context, *Message) error { return h.cancelErr }

func TestConsumeCancel(t *testing.T) {
	ctx := context.Background()
	message := &Message{}
	handleErr := errors.New("handle failed")
	cancelErr := errors.New("cancel failed")

	tests := []struct {
		name      string
		subscribe IConsumeHandle
		want      error
		contains  []error
	}{
		{
			name:      "without cancel handler",
			subscribe: DefaultHandle{},
			want:      handleErr,
		},
		{
			name:      "nil cancel error",
			subscribe: cancelTestHandle{cancelErr: ErrNilCancel},
			want:      handleErr,
		},
		{
			name:      "cancel error",
			subscribe: cancelTestHandle{cancelErr: cancelErr},
			contains:  []error{handleErr, cancelErr},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := consumeCancel(ctx, test.subscribe, message, handleErr)
			if test.want != nil {
				if !errors.Is(got, test.want) || got != test.want {
					t.Fatalf("consumeCancel() = %v, want %v", got, test.want)
				}
				return
			}
			for _, want := range test.contains {
				if !errors.Is(got, want) {
					t.Errorf("consumeCancel() = %v, want to contain %v", got, want)
				}
			}
		})
	}
}
