package util

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

// TODO: Initialize a structured logger like zap or logrus.
// For now, provide basic logging wrappers if needed, or direct use of "log" package is fine.

// Example: A global logger instance (not best practice for libraries, but simple for an app root)
var Logger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

// LogConfig holds configuration for logging
type LogConfig struct {
	ToStderr  bool   // Log to stderr
	ToFile    bool   // Log to file
	FilePath  string // Path to log file
	WithTime  bool   // Include timestamps
	WithLevel bool   // Include log levels
}

// SetupLogging configures the logger based on the provided config
func SetupLogging(cfg LogConfig) (func(), error) {
	var writers []io.Writer
	var cleanup []func()

	// Default format flags
	flags := 0
	if cfg.WithTime {
		flags |= log.Ldate | log.Ltime
	}
	if cfg.WithLevel {
		flags |= log.Lmsgprefix
	}

	// Add stderr writer if requested
	if cfg.ToStderr {
		writers = append(writers, os.Stderr)
	}

	// Add file writer if requested
	var logFile *os.File
	if cfg.ToFile && cfg.FilePath != "" {
		// Create directory if it doesn't exist
		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		var err error
		logFile, err = os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}

		writers = append(writers, logFile)
		cleanup = append(cleanup, func() {
			logFile.Close()
		})
	}

	// Create multi-writer if we have multiple destinations
	var writer io.Writer
	if len(writers) > 1 {
		writer = io.MultiWriter(writers...)
	} else if len(writers) == 1 {
		writer = writers[0]
	} else {
		// No writers specified, use discard
		writer = io.Discard
	}

	// Set the log output and flags
	log.SetOutput(writer)
	log.SetFlags(flags)

	// Return cleanup function
	return func() {
		for _, cleanupFn := range cleanup {
			cleanupFn()
		}
	}, nil
}

func Infof(format string, v ...interface{}) {
	Logger.Printf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	// Could use a different prefix or output for errors
	Logger.Printf("ERROR: "+format, v...)
}
