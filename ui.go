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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/internal/boptions"
	"github.com/retail-ai-inc/beanq/v4/internal/routers"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	uiQueueReportInterval = 10 * time.Second
	uiReadHeaderTimeout   = 5 * time.Second
	uiReadTimeout         = 15 * time.Second
	uiWriteTimeout        = 15 * time.Second
	uiIdleTimeout         = 30 * time.Second
)

//go:embed ui
var views embed.FS

func (c *Client) ServeHTTP(ctx context.Context) error {
	return c.serveHTTP(ctx)
}

func (c *Client) serveHTTP(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.broker == nil || c.config == nil {
		return berror.ErrInvalidConfig.WithMessage("ui requires an initialized client")
	}

	files, err := staticFileInfo(views)
	if err != nil {
		return err
	}

	workflowMongoCollection, disconnectWorkflowMongo, err := c.workflowMongoCollection(ctx)
	if err != nil {
		return err
	}
	defer disconnectWorkflowMongo()

	rlist, err := routers.RedisRouterList(
		views,
		files,
		c.driver,
		c.historyMongoStore(),
		workflowMongoCollection,
		c.config.Redis.Prefix,
		c.config.UI,
	)
	if err != nil {
		return berror.ErrUnsupportedBroker.WithMessage(err.Error())
	}

	addr, err := uiListenAddr(c.config.UI.Port)
	if err != nil {
		return err
	}
	logger.New().Info("Beanq UI Start on port", addr)

	reporterCtx, stopReporter := context.WithCancel(ctx)
	defer stopReporter()
	c.startUIQueueReporter(reporterCtx)

	return runUIServer(ctx, newUIServer(addr, rlist.Mux), c.gracefulShutdownTimeout())
}

func (c *Client) startUIQueueReporter(ctx context.Context) {
	admin, ok := c.broker.(adminReporter)
	if !ok {
		return
	}

	go runUIQueueReporter(ctx, admin, uiQueueReportInterval)
}

func runUIQueueReporter(ctx context.Context, admin adminReporter, interval time.Duration) {
	report := func() {
		if err := admin.QueueMessage(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.New().Error(err)
		}
	}

	report()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			report()
		}
	}
}

func (c *Client) historyMongoStore() *bmongo.BMongo {
	config := c.config
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
		mongoCfg.collectionNames(),
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
	config := c.config
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), c.gracefulShutdownTimeout())
		defer cancel()
		if err := client.Disconnect(shutdownCtx); err != nil {
			logger.New().Error(err)
		}
	}

	collection := client.Database(mongoCfg.Database).Collection(mongoCfg.collectionName("workflow", "workflow_records"))
	return collection, disconnect, nil
}

func newUIServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: uiReadHeaderTimeout,
		ReadTimeout:       uiReadTimeout,
		WriteTimeout:      uiWriteTimeout,
		IdleTimeout:       uiIdleTimeout,
	}
}

func runUIServer(ctx context.Context, server *http.Server, shutdownTimeout time.Duration) error {
	if shutdownTimeout <= 0 {
		shutdownTimeout = boptions.DefaultGracefulShutdownTimeout
	}
	nctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("serve UI: %w", err)
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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		if isUIShutdownTimeout(err) {
			logger.New().Warn("Graceful shutdown timed out; closing server")
			if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
				return fmt.Errorf("force close UI server: %w", closeErr)
			}
			logger.New().Info("Server stopped")
			return nil
		}
		return fmt.Errorf("shut down UI server: %w", err)
	}
	logger.New().Info("Server stopped")
	return nil
}

func isUIShutdownTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

func uiListenAddr(port string) (string, error) {
	normalized := strings.TrimPrefix(strings.TrimSpace(port), ":")
	value, err := strconv.Atoi(normalized)
	if err != nil || value < 1 || value > 65535 {
		return "", fmt.Errorf("invalid UI port %q: must be an integer between 1 and 65535", port)
	}
	return fmt.Sprintf(":%d", value), nil
}

func uiMongoPort(port string) string {
	return fmt.Sprintf(":%s", strings.TrimLeft(port, ":"))
}

func staticFileInfo(source fs.FS) (map[string]time.Time, error) {
	uiFS, err := fs.Sub(source, "ui")
	if err != nil {
		return nil, fmt.Errorf("open embedded UI files: %w", err)
	}
	files := make(map[string]time.Time)

	err = fs.WalkDir(uiFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		files["/"+path] = info.ModTime()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk embedded UI files: %w", err)
	}
	return files, nil
}
