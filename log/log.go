package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Fn typedefs a logging function so it can be passed around
type Fn func(string, ...interface{})

var (
	logger *zap.SugaredLogger
)

func init() {
	// Define our level-handling logic.
	allPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return true
	})

	// Let's also log to stderr
	consoleLog := zapcore.Lock(os.Stderr)
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	// Create our custom loggers
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleLog, allPriority),
	)
	log := zap.New(core)
	logger = log.Sugar()
}

func SetLogger(log *zap.SugaredLogger) {
	logger = log
}

func getLogger() *zap.SugaredLogger {
	return logger
}

func Debugf(format string, args ...interface{}) {
	getLogger().Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	getLogger().Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	getLogger().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	getLogger().Errorf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	getLogger().Panicf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	getLogger().Fatalf(format, args...)
}

func Debug(args ...interface{}) {
	getLogger().Debug(args...)
}

func Info(args ...interface{}) {
	getLogger().Info(args...)
}

func Warn(args ...interface{}) {
	getLogger().Warn(args...)
}

func Error(args ...interface{}) {
	getLogger().Error(args...)
}

func Panic(args ...interface{}) {
	getLogger().Panic(args...)
}

func Fatal(args ...interface{}) {
	getLogger().Fatal(args...)
}

func With(args ...interface{}) *zap.SugaredLogger {
	return getLogger().With(args...)
}
