package config

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapConfig returns the zap logger configuration.
func ZapConfig() *zap.Logger {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
}

// ZapTestConfig returns the zap logger configuration for tests.
func ZapTestConfig() *zap.Logger {
	logger, err := zap.NewProduction()

	if err != nil {
		panic(err.Error())
	}

	return logger
}
