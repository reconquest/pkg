package log

import (
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/cog"
	"github.com/reconquest/karma-go"
)

var (
	logger *cog.Logger
	stderr *lorg.Log
)

type (
	Level = lorg.Level
)

const (
	LevelFatal   = lorg.LevelFatal
	LevelError   = lorg.LevelError
	LevelWarning = lorg.LevelWarning
	LevelInfo    = lorg.LevelInfo
	LevelDebug   = lorg.LevelDebug
	LevelTrace   = lorg.LevelTrace
)

func init() {
	stderr = lorg.NewLog()
	stderr.SetIndentLines(true)
	stderr.SetFormat(lorg.NewFormat(
		`${time:2006-01-02 15:04:05.000} ${level:%s:left:true} ${prefix}%s`,
	))

	logger = cog.NewLogger(stderr)
	logger.SetLevel(lorg.LevelInfo)
}

func SetLevel(level Level) {
	stderr.SetLevel(level)
}

func GetLevel() Level {
	return stderr.GetLevel()
}

func NewChild() *cog.Logger {
	return logger.NewChild()
}

func GetLogger() *cog.Logger {
	return logger
}

func NewChildWithPrefix(prefix string) *cog.Logger {
	return logger.NewChildWithPrefix(prefix)
}

func Fatalf(
	err error,
	message string,
	args ...interface{},
) {
	logger.Fatalf(err, message, args...)
}

func Errorf(
	err error,
	message string,
	args ...interface{},
) {
	logger.Errorf(err, message, args...)
}

func Warningf(
	err error,
	message string,
	args ...interface{},
) {
	logger.Warningf(err, message, args...)
}

func Infof(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	logger.Infof(context, message, args...)
}

func Debugf(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	logger.Debugf(context, message, args...)
}

func Tracef(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	logger.Tracef(context, message, args...)
}

func Fatal(values ...interface{}) {
	logger.Fatal(values...)
}

func Error(values ...interface{}) {
	logger.Error(values...)
}

func Warning(values ...interface{}) {
	logger.Warning(values...)
}

func Info(values ...interface{}) {
	logger.Info(values...)
}

func Debug(values ...interface{}) {
	logger.Debug(values...)
}

func Trace(values ...interface{}) {
	logger.Trace(values...)
}
