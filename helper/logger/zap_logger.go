package logger

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cast"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type (
	ZapLogger struct {
		logger    *zap.Logger
		zapFields []zap.Field
	}
	ZapLoggerConfig struct {
		Filename                    string
		Level                       zapcore.Level
		EncoderType                 string
		MaxSize, MaxAge, MaxBackups int
		LocalTime, Compress         bool
		Pre                         string
	}
)

var (
	logOnce sync.Once
	lg      *ZapLogger

	// set lumberjack logger default parameter
	defaultZapConfig = ZapLoggerConfig{
		Filename:    "",
		Level:       zap.InfoLevel,
		EncoderType: "json",
		MaxSize:     0,
		MaxAge:      0,
		MaxBackups:  0,
		LocalTime:   false,
		Compress:    false,
		Pre:         "beanq",
	}
)

// Logger init logger
func New() *ZapLogger {
	return NewWithConfig(defaultZapConfig)
}

func NewWithConfig(cfg ZapLoggerConfig) *ZapLogger {

	logOnce.Do(func() {
		var (
			encoder zapcore.Encoder
			syncer  zapcore.WriteSyncer
		)

		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = zapcore.RFC3339TimeEncoder
		config.EncodeLevel = zapcore.CapitalLevelEncoder
		config.TimeKey = "time"

		// set encoder
		if cfg.EncoderType == "" {
			cfg.EncoderType = "json"
		}
		switch cfg.EncoderType {
		case "json":
			encoder = zapcore.NewJSONEncoder(config)
		case "console":
			encoder = zapcore.NewConsoleEncoder(config)
		default:
			encoder = zapcore.NewJSONEncoder(config)
		}

		// set level
		level := zapcore.LevelOf(cfg.Level)

		if cfg.Filename == "" {
			syncer = zapcore.WriteSyncer(os.Stdout)
		} else {
			syncer = zapcore.AddSync(&lumberjack.Logger{
				Filename:   cfg.Filename,
				MaxSize:    cfg.MaxSize,
				MaxAge:     cfg.MaxAge,
				MaxBackups: cfg.MaxBackups,
				LocalTime:  cfg.LocalTime,
				Compress:   cfg.Compress,
			})
		}

		levelAble := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level >= zap.ErrorLevel
		})

		newMultiWriteSyncer := zapcore.NewMultiWriteSyncer(syncer)
		newCore := zapcore.NewCore(encoder, newMultiWriteSyncer, level)
		newTee := zapcore.NewTee(newCore)

		l := zap.New(newTee).With(zap.String("pre", cfg.Pre)).WithOptions(zap.AddStacktrace(levelAble))

		lg = &ZapLogger{
			logger:    l,
			zapFields: []zap.Field{},
		}
	})
	return lg
}

func (t ZapLogger) With(key string, val any) ZapLogger {

	switch v := val.(type) {
	case int, int64, *int, *int64:
		t.zapFields = append(t.zapFields, zap.Int64(key, cast.ToInt64(v)))
	case int8, *int8:
		t.zapFields = append(t.zapFields, zap.Int8(key, cast.ToInt8(v)))
	case int16, *int16:
		t.zapFields = append(t.zapFields, zap.Int16(key, cast.ToInt16(v)))
	case int32, *int32:
		t.zapFields = append(t.zapFields, zap.Int32(key, cast.ToInt32(v)))

	case uint, uint64, *uint, *uint64:
		t.zapFields = append(t.zapFields, zap.Uint64(key, cast.ToUint64(v)))
	case uint8, *uint8:
		t.zapFields = append(t.zapFields, zap.Uint8(key, cast.ToUint8(v)))
	case uint16, *uint16:
		t.zapFields = append(t.zapFields, zap.Uint16(key, cast.ToUint16(v)))
	case uint32, *uint32:
		t.zapFields = append(t.zapFields, zap.Uint32(key, cast.ToUint32(v)))

	case uintptr:
		t.zapFields = append(t.zapFields, zap.Uintptr(key, v))
	case *uintptr:
		t.zapFields = append(t.zapFields, zap.Uintptrp(key, v))

	case string:
		t.zapFields = append(t.zapFields, zap.String(key, v))
	case *string:
		t.zapFields = append(t.zapFields, zap.Stringp(key, v))

	case error:
		t.zapFields = append(t.zapFields, zap.Error(v))
	default:

	}
	return t
}

func (t ZapLogger) Info(i ...any) {

	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Info(fmt.Sprint(i...))
	return
}

func (t ZapLogger) Debug(i ...any) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Debug(fmt.Sprint(i...))
	return
}

func (t ZapLogger) Warn(i ...any) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Warn(fmt.Sprint(i...))
	return
}

func (t ZapLogger) Error(i ...any) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Error(fmt.Sprint(i...))
	return
}

func (t ZapLogger) DPanic(i ...any) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).DPanic(fmt.Sprint(i...))
	return
}

func (t ZapLogger) Panic(i ...any) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Panic(fmt.Sprint(i...))
	return
}

func (t ZapLogger) Fatal(i ...any) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Fatal(fmt.Sprint(i...))
	return
}
