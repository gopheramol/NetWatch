// Package utils holds small cross-cutting helpers (logging, id generation)
// shared by every other internal package.
package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger builds a Zap logger writing structured JSON logs to stdout and,
// optionally, to a file at path. level is one of debug/info/warn/error.
func NewLogger(level, path string) (*zap.Logger, error) {
	zapLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		zapLevel = zapcore.InfoLevel
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	encoder := zapcore.NewJSONEncoder(encoderCfg)

	writers := []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}
	if path != "" {
		if dir := filepath.Dir(path); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("creating log directory: %w", err)
			}
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("opening log file: %w", err)
		}
		writers = append(writers, zapcore.AddSync(f))
	}

	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writers...), zapLevel)
	logger := zap.New(core, zap.AddCaller())
	return logger, nil
}
