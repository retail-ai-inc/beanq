package bredis

import (
	"context"
	"sync"

	"golang.org/x/sync/singleflight"
)

type metadataValidationCache struct {
	verified sync.Map
	group    singleflight.Group
}

func (c *metadataValidationCache) ensure(ctx context.Context, key, canonical string, validate func(context.Context) error) error {
	if c == nil {
		return validate(ctx)
	}
	cacheKey := key + "\x00" + canonical
	if _, ok := c.verified.Load(cacheKey); ok {
		return nil
	}
	_, err, _ := c.group.Do(cacheKey, func() (any, error) {
		if _, ok := c.verified.Load(cacheKey); ok {
			return nil, nil
		}
		if err := validate(ctx); err != nil {
			return nil, err
		}
		c.verified.Store(cacheKey, struct{}{})
		return nil, nil
	})
	return err
}
