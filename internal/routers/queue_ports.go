package routers

import "context"

type publishQueue interface {
	Publish(ctx context.Context, data map[string]any) error
}
