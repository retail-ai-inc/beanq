package beanq

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/timex"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/retail-ai-inc/beanq/v4/internal/routers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//go:embed ui
var views embed.FS

func (c *Client) ServeHttp(ctx context.Context) {

	files, err := StaticFileInfo(views)
	if err != nil {
		logger.New().Error(err)
		capture.System.When(c.broker.captureConfig).Then(err)
	}

	go func() {
		timer := timex.TimerPool.Get(10 * time.Second)
		defer timer.Stop()

		for range timer.C {
			select {
			case <-ctx.Done():
				return
			default:

			}
			timer.Reset(10 * time.Second)
			if err := c.broker.tool.QueueMessage(ctx); err != nil {
				logger.New().Error(err)
			}
		}
	}()
	// compatible with unmodified env.json
	httpport := strings.TrimLeft(c.broker.config.UI.Port, ":")
	httpport = fmt.Sprintf(":%s", httpport)

	if err := os.Setenv("GODEBUG", "httpmuxgo122=1"); err != nil {
		logger.New().Error("Error setting environment variables")
		capture.System.When(c.broker.captureConfig).Then(err)
	}

	history := c.broker.config.History
	var mog *bmongo.BMongo
	if history.On {

		// compatible with unmodified env.json
		mongoPort := strings.TrimLeft(history.Mongo.Port, ":")
		mongoPort = fmt.Sprintf(":%s", mongoPort)

		mog = bmongo.NewMongo(
			history.Mongo.Host,
			mongoPort,
			history.Mongo.UserName,
			history.Mongo.Password,
			history.Mongo.Database,
			history.Mongo.Collections,
			history.Mongo.ConnectTimeOut,
			history.Mongo.MaxConnectionPoolSize,
			history.Mongo.MaxConnectionLifeTime,
		)
	}

	var workflowMongoCollection *mongo.Collection
	workflowRecordCfg := c.broker.config.Workflow.Record
	if workflowRecordCfg.On && workflowRecordCfg.Mongo != nil && workflowRecordCfg.Mongo.Database != "" {
		connURI := "mongodb://" + workflowRecordCfg.Mongo.Host + ":" + workflowRecordCfg.Mongo.Port
		opts := options.Client().
			ApplyURI(connURI).
			SetConnectTimeout(workflowRecordCfg.Mongo.ConnectTimeOut).
			SetMaxPoolSize(workflowRecordCfg.Mongo.MaxConnectionPoolSize).
			SetMaxConnIdleTime(workflowRecordCfg.Mongo.MaxConnectionLifeTime)

		if workflowRecordCfg.Mongo.UserName != "" && workflowRecordCfg.Mongo.Password != "" {
			opts.SetAuth(options.Credential{
				AuthSource: workflowRecordCfg.Mongo.Database,
				Username:   workflowRecordCfg.Mongo.UserName,
				Password:   workflowRecordCfg.Mongo.Password,
			})
		}

		client, err := mongo.Connect(ctx, opts)
		if err != nil {
			panic(err)
		}
		workflowMongoCollection = client.Database(workflowRecordCfg.Mongo.Database).Collection(workflowRecordCfg.Mongo.Collection)
	}

	rlist := routers.RouterList(
		views,
		files,
		c.broker.client.(redis.UniversalClient),
		mog, workflowMongoCollection,
		c.broker.config.Redis.Prefix, c.broker.config.UI)

	logger.New().Info("Beanq UI Start on port", httpport)

	server := &http.Server{
		Addr:         httpport,
		Handler:      rlist.Mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	nctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			capture.System.When(c.broker.captureConfig).Then(err)
			logger.New().Fatal("Error starting server:", err)
		}
	}()

	<-nctx.Done()
	logger.New().Info("Prepare to shut down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.New().Fatal("Error shutting down server:", err)
	}
	logger.New().Info("Server stopped")
}

func StaticFileInfo(fs2 fs.FS) (map[string]time.Time, error) {

	files := make(map[string]time.Time, 0)

	err := fs.WalkDir(fs2, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			arr := strings.SplitAfter(path, "ui")
			if len(arr) == 2 {
				info, _ := d.Info()
				files[arr[1]] = info.ModTime()
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}
