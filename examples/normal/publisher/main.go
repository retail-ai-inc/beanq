package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

var (
	configOnce sync.Once
	bqConfig   beanq.BeanqConfig
)

func initCnf() *beanq.BeanqConfig {
	configOnce.Do(func() {
		var envPath string = "./"
		if _, file, _, ok := runtime.Caller(0); ok {
			envPath = filepath.Dir(file)
		}

		vp := viper.New()
		vp.AddConfigPath(envPath)
		vp.SetConfigType("json")
		vp.SetConfigName("env")

		if err := vp.ReadInConfig(); err != nil {
			log.Fatalf("Unable to open beanq env.json file: %v", err)
		}

		// IMPORTANT: Unmarshal the env.json into global Config object.
		if err := vp.Unmarshal(&bqConfig); err != nil {
			log.Fatalf("Unable to unmarshal the beanq env.json file: %v", err)
		}
	})
	return &bqConfig
}

type Str struct {
	Id     uint64
	Values map[string]any
}

type SortStr []Str

func (t SortStr) Len() int {
	return len(t)
}
func (t SortStr) Less(i, j int) bool {
	return t[i].Id < t[j].Id
}
func (t SortStr) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func main() {
	sl := make([]Str, 0)
	sl = append(sl, Str{
		Id:     11,
		Values: nil,
	}, Str{
		Id:     2,
		Values: nil,
	}, Str{
		Id:     3,
		Values: nil,
	})
	sort.Sort(SortStr(sl))
	fmt.Printf("%+v \n", sl)
	return
	pubMoreAndPriorityInfo()
}

func pubMoreAndPriorityInfo() {
	pub := beanq.New(initCnf())
	m := make(map[string]string)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		if err := pub.BQ().WithContext(ctx).Publish("default-channel", "default-topic", b); err != nil {
			logger.New().Error(err)
		}
	}
}
