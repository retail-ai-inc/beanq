package workflow

import (
	"context"
	"errors"

	"github.com/retail-ai-inc/beanq"
)

type handler struct {
	gid string
	msg *beanq.Message
	fn  func(ctx context.Context, id string) (*beanq.Message, error)
}

func (h *handler) execute(ctx context.Context) error {
	msg, err := h.fn(ctx, h.gid)
	h.msg = msg
	if err != nil {
		return err
	}
	// need more logic here to determine whether rollback is needed
	if msg.ExecutionStatus == "failed" {
		return errors.New("need rollback")
	}
	return nil
}

func (h *handler) rollback(ctx context.Context) error {
	// set needRollback is true and resend the message to consumer, so consumer will receive the rollback instruction.
	h.msg.NeedRollback = true
	_, err := h.fn(ctx, h.gid)
	if err != nil {
		return err
	}
	return nil
}

type Pipe struct {
	rollbackHandlers []handler
	handlers         []handler
}

func NewPipe() *Pipe {
	return &Pipe{}
}

func (w *Pipe) AddCargo(id string, h func(ctx context.Context, id string) (*beanq.Message, error)) *Pipe {
	w.handlers = append(w.handlers, handler{
		gid: id,
		fn:  h,
	})
	return w
}

func (w *Pipe) ExecuteWithContext(ctx context.Context) error {
	for _, h := range w.handlers {
		err := h.execute(ctx)
		if err != nil {
			// rollback here
			for _, rh := range w.rollbackHandlers {
				err := rh.rollback(ctx)
				if err != nil {
					// return the rollback error
					return err
				}
			}
			// should return here
			return err
		}
		w.rollbackHandlers = append(w.rollbackHandlers, h)
	}
	w.clear()
	return nil
}

func (w *Pipe) clear() {
	w.rollbackHandlers = nil
	w.handlers = nil
}
