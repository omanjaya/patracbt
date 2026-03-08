package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger

func Init(env string) {
	var cfg zap.Config

	if env == "production" {
		cfg = zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	cfg.OutputPaths = []string{"stdout"}

	zapLogger, err := cfg.Build()
	if err != nil {
		os.Exit(1)
	}

	Log = zapLogger.Sugar()
}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
