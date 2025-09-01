package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(serviceName, environment string) error {
	config := zap.NewProductionConfig()

	// Spring Boot slf4j 스타일 필드 설정
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.CallerKey = "logger"
	config.EncoderConfig.StacktraceKey = "stacktrace"

	// 개발 환경에서는 콘솔 포맷 사용
	if environment == "prod" {
		config.Encoding = "json"
		config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	} else {
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// 로그 레벨 설정
	if environment == "prod" {
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	} else {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	// 서비스 정보를 초기 필드로 추가
	config.InitialFields = map[string]interface{}{
		"service": serviceName,
		"env":     environment,
	}

	logger, err := config.Build(zap.AddCaller())
	if err != nil {
		return err
	}

	Logger = logger
	return nil
}

// slf4j 스타일 래퍼 함수들

func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}

// 구조화된 로깅을 위한 헬퍼 함수들

func WithError(err error) zap.Field {
	return zap.Error(err)
}

func WithString(key, value string) zap.Field {
	return zap.String(key, value)
}

func WithInt(key string, value int) zap.Field {
	return zap.Int(key, value)
}

func WithInt64(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

func WithDuration(key string, value interface{}) zap.Field {
	switch v := value.(type) {
	case int64:
		return zap.Duration(key, time.Duration(v))
	case time.Duration:
		return zap.Duration(key, v)
	default:
		return zap.Any(key, value)
	}
}

func WithAny(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

// 기존 log 패키지와의 호환성을 위한 함수들

func Printf(format string, args ...interface{}) {
	Logger.Sugar().Infof(format, args...)
}

func Println(args ...interface{}) {
	Logger.Sugar().Info(args...)
}

func Print(args ...interface{}) {
	Logger.Sugar().Info(args...)
}

func Fatalf(format string, args ...interface{}) {
	Logger.Sugar().Fatalf(format, args...)
}

func Fatalln(args ...interface{}) {
	Logger.Sugar().Fatal(args...)
}

// 로그 플러시 및 종료
func Sync() error {
	return Logger.Sync()
}

// 환경 변수에서 로그 설정 초기화
func InitFromEnv(serviceName string) error {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "production"
	}
	return InitLogger(serviceName, env)
}
