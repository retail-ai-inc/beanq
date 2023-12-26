package logger

import (
	"os"
	"sync"

	"github.com/spf13/cast"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type (
	LoggerNew struct {
		logger    *zap.Logger
		zapFields []zap.Field
	}
)

var (
	logOnce sync.Once
	lg      *LoggerNew

	// set lumberjack logger default parameter
	defaultLogger = struct {
		Filename                    string
		MaxSize, MaxAge, MaxBackups int
		LocalTime, Compress         bool
		Pre                         string
	}{
		Filename:   "./log.txt",
		MaxSize:    0,
		MaxAge:     0,
		MaxBackups: 0,
		LocalTime:  false,
		Compress:   false,
		Pre:        "beanq",
	}
)

func NewLogger(fileName string, maxSize, maxAge, maxBackups int, localTime, compress, accessLog bool) *LoggerNew {

	logOnce.Do(func() {
		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = zapcore.RFC3339TimeEncoder
		config.EncodeLevel = zapcore.CapitalLevelEncoder
		config.TimeKey = "time"
		cfg := zapcore.NewJSONEncoder(config)

		level := zapcore.LevelOf(zap.InfoLevel)

		if fileName == "" {
			fileName = defaultLogger.Filename
		}
		if maxSize < 0 {
			maxSize = defaultLogger.MaxSize
		}
		if maxAge < 0 {
			maxAge = defaultLogger.MaxAge
		}
		if maxBackups < 0 {
			maxBackups = defaultLogger.MaxBackups
		}
		syncer := &lumberjack.Logger{
			Filename:   fileName,
			MaxSize:    maxSize,
			MaxAge:     maxAge,
			MaxBackups: maxBackups,
			LocalTime:  localTime,
			Compress:   compress,
		}

		levelAble := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level >= zap.ErrorLevel
		})
		var isLog []zapcore.WriteSyncer
		if accessLog {
			isLog = []zapcore.WriteSyncer{os.Stdout, zapcore.AddSync(syncer)}
		}
		newMultiWriteSyncer := zapcore.NewMultiWriteSyncer(isLog...)
		newCore := zapcore.NewCore(cfg, newMultiWriteSyncer, level)
		newTee := zapcore.NewTee(newCore)

		l := zap.New(newTee).With(zap.String("pre", defaultLogger.Pre)).WithOptions(zap.AddStacktrace(levelAble))

		lg = &LoggerNew{
			logger:    l,
			zapFields: []zap.Field{},
		}
	})
	return lg
}

func (t LoggerNew) With(key string, val any) LoggerNew {

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

func (t LoggerNew) Info(msg string) {

	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Info(msg)
	return
}

func (t LoggerNew) Debug(msg string) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Debug(msg)
	return
}

func (t LoggerNew) Warn(msg string) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Warn(msg)
	return
}

func (t LoggerNew) Error(msg string) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Error(msg)
	return
}

func (t LoggerNew) DPanic(msg string) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).DPanic(msg)
	return
}

func (t LoggerNew) Panic(msg string) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Panic(msg)
	return
}

func (t LoggerNew) Fatal(msg string) {
	defer func() {
		_ = t.logger.Sync()
	}()
	t.logger.With(t.zapFields...).Fatal(msg)
	return
}
