package routers

import "github.com/retail-ai-inc/beanq/v3/helper/bwebframework"

type Index struct {
}

func NewIndex() *Index {
	return &Index{}
}

func (t *Index) Home(ctx *bwebframework.BeanContext) error {
	return nil
}
