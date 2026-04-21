package mudlog

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/natefinch/lumberjack"
)

var (
	slogInstance *slog.Logger
	logLevel     = new(slog.LevelVar) // goroutine safe way to change log levels
)

type teeLogger interface {
	Println(level string, v ...any)
}

func SetLogLevel(lvl string) {

	if len(lvl) > 0 {
		if lvl[0:1] == `M` {
			logLevel.Set(slog.LevelInfo)
			return
		} else if lvl[0:1] == `L` {
			logLevel.Set(slog.LevelWarn)
			return
		}
	}

	logLevel.Set(slog.LevelDebug)

}

func SetupLogger(inGameLogger teeLogger, logLevel string, logPath string, colorLogs bool) {

	SetLogLevel(strings.ToUpper(logLevel))

	// No filepath? Write to Stderr.
	if logPath == `` {
		slogInstance = slog.New(
			getLogHandler(os.Stderr, inGameLogger, colorLogs),
		)
		return
	}

	// Setup file logging
	fileInfo, err := os.Stat(logPath)
	if err == nil {
		if fileInfo.IsDir() {
			log.Fatalf("log file path is a directory: %s", logPath)
		}

	} else if os.IsNotExist(err) {
		// File does not exist; check if the directory exists
		dir := filepath.Dir(logPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			log.Fatalf("directory for log file does not exist: %s", dir)
		}
	} else {
		// Some other error
		log.Fatalf("error accessing log file path: %v", err)
	}

	lj := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100,  // Maximum size in megabytes before rotation
		MaxBackups: 10,   // Maximum number of old log files to retain
		Compress:   true, // Compress rotated files
	}

	// Write to both terminal and file
	multiOut := io.MultiWriter(os.Stderr, lj)

	slogInstance = slog.New(
		getLogHandler(multiOut, inGameLogger, colorLogs),
	)

}

func Debug(msg string, args ...any) {
	slogInstance.Log(context.Background(), slog.LevelDebug, msg, args...)
}

func Info(msg string, args ...any) {
	slogInstance.Log(context.Background(), slog.LevelInfo, msg, args...)
}

func Warn(msg string, args ...any) {
	slogInstance.Log(context.Background(), slog.LevelWarn, msg, args...)
}

func Error(msg string, args ...any) {
	slogInstance.Log(context.Background(), slog.LevelError, msg, args...)
}

// HasLogger returns true if the logger has been initialized via SetupLogger.
// Used to guard logging calls in functions that may be called during tests.
func HasLogger() bool {
	return slogInstance != nil
}
