package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggerParameter struct {
	InfoFileName        string
	ErrFileName         string
	MaxSize             int
	MaxAge              int
	MaxBackups          int
	LocalTime, Compress bool
}
type LoggerInfoFun func(info *LoggerParameter)

func WithInfoFile(file string) LoggerInfoFun {
	return func(info *LoggerParameter) {
		info.InfoFileName = file
	}
}
func WithErrFile(file string) LoggerInfoFun {
	return func(info *LoggerParameter) {
		info.ErrFileName = file
	}
}
func WithMaxSize(size int) LoggerInfoFun {
	return func(info *LoggerParameter) {
		info.MaxSize = size
	}
}
func WithMaxAge(age int) LoggerInfoFun {
	return func(info *LoggerParameter) {
		info.MaxAge = age
	}
}
func WithMaxBackups(backup int) LoggerInfoFun {
	return func(info *LoggerParameter) {
		info.MaxBackups = backup
	}
}
func WithLocalTime(b bool) LoggerInfoFun {
	return func(info *LoggerParameter) {
		info.LocalTime = b
	}
}
func WithCompress(b bool) LoggerInfoFun {
	return func(info *LoggerParameter) {
		info.Compress = b
	}
}
func composeParameter(funs ...LoggerInfoFun) *LoggerParameter {

	param := LoggerParameter{
		InfoFileName: "",
		ErrFileName:  "",
		MaxSize:      200,
		MaxAge:       3,
		MaxBackups:   5,
		LocalTime:    false,
		Compress:     false,
	}
	for _, f := range funs {
		f(&param)
	}
	return &param
}

func InitLogger(funs ...LoggerInfoFun) *zap.Logger {

	highPriority := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zap.ErrorLevel
	})
	lowerPriority := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level < zap.ErrorLevel && level >= zap.DebugLevel
	})

	// The log will be print on console by default
	arr, highArr := []zapcore.WriteSyncer{os.Stdout}, []zapcore.WriteSyncer{os.Stdout}
	// and save the log in a file

	parameter := composeParameter(funs...)
	if parameter.InfoFileName != "" {
		arr = append(arr, getInfoWriter(parameter))
	}
	if parameter.ErrFileName != "" {
		highArr = append(highArr, getErrWriter(parameter))
	}

	infoFile := zapcore.NewCore(encoder(), zapcore.NewMultiWriteSyncer(arr...), lowerPriority)
	errFile := zapcore.NewCore(encoder(), zapcore.NewMultiWriteSyncer(highArr...), highPriority)
	logger := zap.New(zapcore.NewTee(infoFile, errFile), zap.AddCaller())
	defer logger.Sync()
	return logger
}

// log format
func encoder() zapcore.Encoder {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeLevel = zapcore.CapitalLevelEncoder
	config.TimeKey = "time"
	return zapcore.NewJSONEncoder(config)
}

func getInfoWriter(parameter *LoggerParameter) zapcore.WriteSyncer {

	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   parameter.InfoFileName,
		MaxSize:    parameter.MaxSize,
		MaxAge:     parameter.MaxAge,
		MaxBackups: parameter.MaxBackups,
		LocalTime:  parameter.LocalTime,
		Compress:   parameter.Compress,
	})
}
func getErrWriter(parameter *LoggerParameter) zapcore.WriteSyncer {

	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   parameter.ErrFileName,
		MaxSize:    parameter.MaxSize,
		MaxAge:     parameter.MaxAge,
		MaxBackups: parameter.MaxBackups,
		LocalTime:  parameter.LocalTime,
		Compress:   parameter.Compress,
	})
}
