package beanq

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/retail-ai-inc/beanq/v4/internal/routers"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	uiQueueReportInterval = 10 * time.Second
	uiShutdownTimeout     = 10 * time.Second
	uiReadTimeout         = 15 * time.Second
	uiWriteTimeout        = 15 * time.Second
	uiIdleTimeout         = 30 * time.Second
)

//go:embed ui
var views embed.FS

func (c *Client) ServeHttp(ctx context.Context) {
	if err := c.serveHTTP(ctx); err != nil {
		logger.New().Error(err)
		if c != nil && c.broker != nil {
			capture.System.When(c.broker.captureConfig).Then(err)
		}
	}
}

func (c *Client) serveHTTP(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.broker == nil || c.broker.config == nil {
		return ErrInvalidConfig.WithMessage("ui requires an initialized client")
	}

	files, err := StaticFileInfo(views)
	if err != nil {
		return err
	}

	c.startUIQueueReporter(ctx)

	if err := os.Setenv("GODEBUG", "httpmuxgo122=1"); err != nil {
		return fmt.Errorf("set GODEBUG: %w", err)
	}

	workflowMongoCollection, disconnectWorkflowMongo, err := c.workflowMongoCollection(ctx)
	if err != nil {
		return err
	}
	defer disconnectWorkflowMongo()

	redisClient, ok := c.broker.client.(redis.UniversalClient)
	if !ok {
		return ErrUnsupportedBroker.WithMessage("ui requires redis client")
	}

	rlist := routers.RouterList(
		views,
		files,
		redisClient,
		c.historyMongoStore(),
		workflowMongoCollection,
		c.broker.config.Redis.Prefix,
		c.broker.config.UI,
	)

	addr := uiListenAddr(c.broker.config.UI.Port)
	logger.New().Info("Beanq UI Start on port", addr)

	return runUIServer(ctx, newUIServer(addr, rlist.Mux))
}

func (c *Client) startUIQueueReporter(ctx context.Context) {
	if c.broker == nil || c.broker.tool == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(uiQueueReportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.broker.tool.QueueMessage(ctx); err != nil {
					logger.New().Error(err)
				}
			}
		}
	}()
}

func (c *Client) historyMongoStore() *bmongo.BMongo {
	config := c.broker.config
	mongoCfg := config.Mongo
	if !config.History.On || mongoCfg == nil {
		return nil
	}

	return bmongo.NewMongo(
		mongoCfg.Host,
		uiMongoPort(mongoCfg.Port),
		mongoCfg.UserName,
		mongoCfg.Password,
		mongoCfg.Database,
		uiCollectionNames(mongoCfg),
		mongoCfg.ConnectTimeOut,
		mongoCfg.MaxConnectionPoolSize,
		mongoCfg.MaxConnectionLifeTime,
		bmongo.MongoSSLConfig{
			On:     mongoCfg.SSL.On,
			CAFile: mongoCfg.SSL.CAFile,
			Verify: mongoCfg.SSL.Verify,
		},
	)
}

func (c *Client) workflowMongoCollection(ctx context.Context) (*mongo.Collection, func(), error) {
	config := c.broker.config
	mongoCfg := config.Mongo
	if !config.WorkFlow.On || mongoCfg == nil || mongoCfg.Database == "" {
		return nil, func() {}, nil
	}

	opts, err := mongoClientOptions(mongoCfg)
	if err != nil {
		return nil, nil, err
	}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, nil, err
	}

	disconnect := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), uiShutdownTimeout)
		defer cancel()
		if err := client.Disconnect(shutdownCtx); err != nil {
			logger.New().Error(err)
		}
	}

	collection := client.Database(mongoCfg.Database).Collection(uiCollectionName(mongoCfg, "workflow", "workflow_records"))
	return collection, disconnect, nil
}

func newUIServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  uiReadTimeout,
		WriteTimeout: uiWriteTimeout,
		IdleTimeout:  uiIdleTimeout,
	}
}

func runUIServer(ctx context.Context, server *http.Server) error {
	nctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-nctx.Done():
	}

	logger.New().Info("Prepare to shut down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), uiShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		if isUIShutdownTimeout(err) {
			logger.New().Warn("Graceful shutdown timed out; closing server")
			if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
				return closeErr
			}
			logger.New().Info("Server stopped")
			return nil
		}
		return err
	}
	logger.New().Info("Server stopped")
	return nil
}

func isUIShutdownTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

func uiListenAddr(port string) string {
	return fmt.Sprintf(":%s", strings.TrimLeft(port, ":"))
}

func uiMongoPort(port string) string {
	return uiListenAddr(port)
}

func uiCollectionNames(config *Mongo) map[string]string {
	collections := make(map[string]string, len(config.Collections))
	for name, collection := range config.Collections {
		collections[name] = collection.Name
	}
	return collections
}

func uiCollectionName(config *Mongo, key, fallback string) string {
	if config == nil {
		return fallback
	}
	if collection, ok := config.Collections[key]; ok && collection.Name != "" {
		return collection.Name
	}
	return fallback
}

func StaticFileInfo(fs2 fs.FS) (map[string]time.Time, error) {
	files := make(map[string]time.Time, 0)

	err := fs.WalkDir(fs2, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		name, ok := strings.CutPrefix(path, "ui")
		if !ok {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		files[name] = info.ModTime()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
