package logger

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/labstack/gommon/log"
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
		DefaultWriter io.Writer
		Filename      string
		EncoderType   string
		Pre           string
		MaxSize       int
		MaxAge        int
		MaxBackups    int
		Level         zapcore.Level
		LocalTime     bool
		Compress      bool
	}
)

var _ Logger = (*ZapLogger)(nil)

var (
	logOnce sync.Once
	lg      *ZapLogger

	// set lumberjack logger default parameter
	defaultZapConfig = ZapLoggerConfig{
		DefaultWriter: os.Stdout,
		Filename:      "",
		Level:         zap.InfoLevel,
		EncoderType:   "json",
		MaxSize:       0,
		MaxAge:        0,
		MaxBackups:    0,
		LocalTime:     false,
		Compress:      false,
		Pre:           "beanq",
	}
)

// New init logger
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
			syncer = zapcore.WriteSyncer(zapcore.AddSync(cfg.DefaultWriter))
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
		if key == "" {
			t.zapFields = append(t.zapFields, zap.Error(v))
		} else {
			t.zapFields = append(t.zapFields, zap.NamedError(key, v))
		}
	default:
		t.zapFields = append(t.zapFields, zap.Any(key, v))
	}
	return t
}

func (t ZapLogger) Output() io.Writer   { return defaultZapConfig.DefaultWriter }
func (t ZapLogger) SetOutput(io.Writer) {}
func (t ZapLogger) Prefix() string      { return defaultZapConfig.Pre }
func (t ZapLogger) SetPrefix(string)    {}
func (t ZapLogger) Level() log.Lvl      { return log.Lvl(defaultZapConfig.Level + 1) }
func (t ZapLogger) SetLevel(log.Lvl)    {}
func (t ZapLogger) SetHeader(string)    {}

func (t ZapLogger) Print(i ...any)                    { t.Info(i...) }
func (t ZapLogger) Printf(format string, args ...any) { t.Infof(format, args...) }
func (t ZapLogger) Printj(j log.JSON)                 { t.Info(j) }
func (t ZapLogger) Debugf(format string, args ...any) { t.Debug(fmt.Sprintf(format, args...)) }
func (t ZapLogger) Debugj(j log.JSON)                 { t.Debug(j) }
func (t ZapLogger) Infof(format string, args ...any)  { t.Info(fmt.Sprintf(format, args...)) }
func (t ZapLogger) Infoj(j log.JSON)                  { t.Info(j) }
func (t ZapLogger) Warnf(format string, args ...any)  { t.Warn(fmt.Sprintf(format, args...)) }
func (t ZapLogger) Warnj(j log.JSON)                  { t.Warn(j) }
func (t ZapLogger) Errorf(format string, args ...any) { t.Error(fmt.Sprintf(format, args...)) }
func (t ZapLogger) Errorj(j log.JSON)                 { t.Error(j) }
func (t ZapLogger) Fatalf(format string, args ...any) { t.Fatal(fmt.Sprintf(format, args...)) }
func (t ZapLogger) Fatalj(j log.JSON)                 { t.Fatal(j) }
func (t ZapLogger) Panicf(format string, args ...any) { t.Panic(fmt.Sprintf(format, args...)) }
func (t ZapLogger) Panicj(j log.JSON)                 { t.Panic(j) }

func (t ZapLogger) Info(i ...any) {
	t.logger.With(t.zapFields...).Info(fmt.Sprint(i...))
}

func (t ZapLogger) Debug(i ...any) {
	t.logger.With(t.zapFields...).Debug(fmt.Sprint(i...))
}

func (t ZapLogger) Warn(i ...any) {
	t.logger.With(t.zapFields...).Warn(fmt.Sprint(i...))
}

func (t ZapLogger) Error(i ...any) {
	t.logger.With(t.zapFields...).Error(fmt.Sprint(i...))
}

func (t ZapLogger) DPanic(i ...any) {
	t.logger.With(t.zapFields...).DPanic(fmt.Sprint(i...))
}

func (t ZapLogger) Panic(i ...any) {
	t.logger.With(t.zapFields...).Panic(fmt.Sprint(i...))
}

func (t ZapLogger) Fatal(i ...any) {
	t.logger.With(t.zapFields...).Fatal(fmt.Sprint(i...))
}

func (t ZapLogger) Sync() error {
	return t.logger.Sync()
}
