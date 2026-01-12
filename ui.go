package beanq

import (
	"context"
	"embed"
	"fmt"
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

	mux := http.NewServeMux()

	mongoCfg := c.broker.config.Mongo
	collections := make(map[string]string)

	for s, collection := range mongoCfg.Collections {
		collections[s] = collection.Name
	}

	var mog *bmongo.BMongo
	if c.broker.config.History.On {

		// compatible with unmodified env.json
		mongoPort := strings.TrimLeft(mongoCfg.Port, ":")
		mongoPort = fmt.Sprintf(":%s", mongoPort)

		mog = bmongo.NewMongo(
			mongoCfg.Host,
			mongoPort,
			mongoCfg.UserName,
			mongoCfg.Password,
			mongoCfg.Database,
			collections,
			mongoCfg.ConnectTimeOut,
			mongoCfg.MaxConnectionPoolSize,
			mongoCfg.MaxConnectionLifeTime,
		)
	}

	var workflowMongoCollection *mongo.Collection

	collection := "workflow_records"
	if v, ok := mongoCfg.Collections["workflow"]; ok {
		collection = v.Name
	}

	if c.broker.config.WorkFlow.On && mongoCfg != nil && mongoCfg.Database != "" {
		connURI := "mongodb://" + mongoCfg.Host + ":" + mongoCfg.Port
		opts := options.Client().
			ApplyURI(connURI).
			SetConnectTimeout(mongoCfg.ConnectTimeOut).
			SetMaxPoolSize(mongoCfg.MaxConnectionPoolSize).
			SetMaxConnIdleTime(mongoCfg.MaxConnectionLifeTime)

		if mongoCfg.UserName != "" && mongoCfg.Password != "" {
			opts.SetAuth(options.Credential{
				AuthSource: mongoCfg.Database,
				Username:   mongoCfg.UserName,
				Password:   mongoCfg.Password,
			})
		}

		client, err := mongo.Connect(ctx, opts)
		if err != nil {
			panic(err)
		}
		workflowMongoCollection = client.Database(mongoCfg.Database).Collection(collection)
	}

	routers.NewRouters(
		mux,
		views,
		files,
		c.broker.client.(redis.UniversalClient),
		mog, workflowMongoCollection,
		c.broker.config.Redis.Prefix, c.broker.config.UI)

	logger.New().Info("Beanq UI Start on port", httpport)
	server := &http.Server{
		Addr:         httpport,
		Handler:      mux,
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
