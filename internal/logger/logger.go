package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// L is the global logger instance
	L *zap.Logger
	// S is the global sugared logger for convenience
	S *zap.SugaredLogger
)

// Config holds logger configuration
type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json, console
	ServerName string // MCP server name for log context
	Version    string // Server version
}

// Init initializes the global logger with the given configuration
func Init(cfg Config) error {
	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	var encoder zapcore.Encoder

	if cfg.Format == "console" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stderr),
		level,
	)

	// Create logger with context fields
	L = zap.New(core).With(
		zap.String("server", cfg.ServerName),
		zap.String("version", cfg.Version),
	)
	S = L.Sugar()

	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}

// WithRequestID returns a logger with request context
func WithRequestID(requestID string) *zap.Logger {
	return L.With(zap.String("request_id", requestID))
}

// WithTool returns a logger with tool context
func WithTool(requestID, toolName string) *zap.Logger {
	return L.With(
		zap.String("request_id", requestID),
		zap.String("tool", toolName),
	)
}

// WithConnection returns a logger with connection context
func WithConnection(clientAddr, mode string) *zap.Logger {
	return L.With(
		zap.String("client_addr", clientAddr),
		zap.String("mode", mode),
	)
}
