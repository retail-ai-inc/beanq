package routers

import "context"

type enqueueQueue interface {
	Enqueue(ctx context.Context, data map[string]any) error
}
