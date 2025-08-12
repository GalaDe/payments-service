package log

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	Log   *zap.Logger
}

type LoggerInterface interface {
	Logger() *zap.Logger

	Debug(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
	Warn(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})

	SendError(errMessage string, addlContext map[string]interface{}, errorType string)
}

func New(name string) *Logger {
	l := os.Getenv("LOG_LEVEL")

	l = "debug"
	switch strings.ToLower(l) {
	case "debug":
		l = "debug"
	case "info":
		l = "info"
	case "warn":
		l = "warn"
	case "error":
		l = "error"
	case "fatal":
		l = "fatal"
	default:
		l = "info"
	}

	stringCfg := fmt.Sprintf(`{
		"level": "%s",
		"encoding": "json",
		"outputPaths": ["stdout"],
		"errorOutputPaths": ["stderr"],
		"initialFields": {"app_name": "%s"},
		"encoderConfig": {
		  "messageKey": "message",
		  "levelKey": "level",
		  "timeKey": "timestamp",
		  "levelEncoder": "lowercase"
		}
	}`, l, name)
	jsonCfg := []byte(stringCfg)

	var err error
	var cfg zap.Config
	if err := json.Unmarshal(jsonCfg, &cfg); err != nil {
		panic(fmt.Sprintf("FATAL ERROR: loading logger %s", err))
	}
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoder(func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format("2006-01-02T15:04:05Z0700"))
	})
	logger, err := cfg.Build()

	if err != nil {
		panic(fmt.Sprintf("FATAL ERROR: loading logger %s", err))
	}

	defer logger.Sync()

	return &Logger{
		Log:   logger,
	}
}

func (l *Logger) Logger() *zap.Logger {
	return l.Log
}

func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	l.Log.Sugar().Debugf(msg, keyvals...)
}

func (l *Logger) Info(msg string, keyvals ...interface{}) {
	l.Log.Sugar().Infof(msg, keyvals...)
}

func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	l.Log.Sugar().Warnf(msg, keyvals...)
}

func (l *Logger) Error(msg string, keyvals ...interface{}) {
	l.Log.Sugar().Errorf(msg, keyvals...)
}

// SendError is a wrapper error function that logs via zap logger and also sends a notification to our monitoring service (honeybadger)
func (l *Logger) SendError(errMessage string, addlContext map[string]interface{}, errorType string) {
	l.Log.Sugar().Error(errMessage)
}
