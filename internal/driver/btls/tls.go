package btls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
)

func LoadTLSConfigFromCA(caFile string, verifyCertificate bool) (*tls.Config, error) {

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file %q: %w", caFile, err)
	}

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caPEM); !ok {
		return nil, fmt.Errorf("CA file %q does not contain a valid PEM certificate", caFile)
	}

	return &tls.Config{
		RootCAs:            pool,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: verifyCertificate, // Preserve existing Redis verifyCertificate behavior.
	}, nil
}

func WatchCAFile(ctx context.Context, serviceName string, caFile string, reload func(context.Context) error) error {

	absCAFile, err := filepath.Abs(caFile)
	if err != nil {
		return err
	}

	watchDir := filepath.Dir(absCAFile)
	watchBase := filepath.Base(absCAFile)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := watcher.Add(watchDir); err != nil {
		_ = watcher.Close()
		return err
	}

	go func() {
		defer watcher.Close()

		var (
			timer   *time.Timer
			timerMu sync.Mutex
		)

		triggerReload := func(reason string) {
			timerMu.Lock()
			defer timerMu.Unlock()

			if timer != nil {
				timer.Stop()
			}

			timer = time.AfterFunc(500*time.Millisecond, func() {
				reloadCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				logger.New().Debug("CA file changed, reloading client", "service", serviceName, "reason", reason)
				if err := reload(reloadCtx); err != nil {
					logger.New().Error("reload failed, keep using old client", "service", serviceName, "error", err)
				}
			})
		}

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if !isTargetCAEvent(event, watchDir, watchBase) {
					continue
				}

				if event.Has(fsnotify.Write) ||
					event.Has(fsnotify.Create) ||
					event.Has(fsnotify.Rename) ||
					event.Has(fsnotify.Remove) ||
					event.Has(fsnotify.Chmod) {
					triggerReload(event.String())
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.New().Debug("CA watcher error", "service", serviceName, "error", err)
			}
		}
	}()

	logger.New().Debug("watching CA file", "service", serviceName, "file", absCAFile)
	return nil
}

func isTargetCAEvent(event fsnotify.Event, watchDir, watchBase string) bool {

	if filepath.Base(event.Name) == watchBase {
		return true
	}

	// Kubernetes Secret/ConfigMap volumes update symlinks like "..data".
	name := filepath.Base(event.Name)
	if name == "..data" || name == "..data_tmp" {
		return true
	}

	// Some deploy tools replace the whole directory entry with a temp file.
	return filepath.Dir(event.Name) == watchDir && filepath.Base(event.Name) == watchBase
}
