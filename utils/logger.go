package utils

import (
	"exam-system/config"
	"os"

	"github.com/sirupsen/logrus"
)

func InitLogger() *logrus.Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.AppConfig.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set log format
	if config.AppConfig.Logging.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	// Set output
	logger.SetOutput(os.Stdout)

	return logger
}

// LogEntry represents a structured log entry
type LogEntry struct {
	*logrus.Entry
}

// NewLogEntry creates a new log entry with common fields
func NewLogEntry(logger *logrus.Logger) *LogEntry {
	return &LogEntry{
		Entry: logrus.NewEntry(logger),
	}
}

// WithRequestID adds request ID to log entry
func (l *LogEntry) WithRequestID(requestID string) *LogEntry {
	return &LogEntry{
		Entry: l.Entry.WithField("request_id", requestID),
	}
}

// WithUserID adds user ID to log entry
func (l *LogEntry) WithUserID(userID uint) *LogEntry {
	return &LogEntry{
		Entry: l.Entry.WithField("user_id", userID),
	}
}

// WithError adds error to log entry
func (l *LogEntry) WithError(err error) *LogEntry {
	return &LogEntry{
		Entry: l.Entry.WithError(err),
	}
}

// WithFields adds multiple fields to log entry
func (l *LogEntry) WithFields(fields logrus.Fields) *LogEntry {
	return &LogEntry{
		Entry: l.Entry.WithFields(fields),
	}
}

