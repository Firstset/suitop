package util

import (
	"log"
	"os"
)

// TODO: Initialize a structured logger like zap or logrus.
// For now, provide basic logging wrappers if needed, or direct use of "log" package is fine.

// Example: A global logger instance (not best practice for libraries, but simple for an app root)
var Logger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

func Infof(format string, v ...interface{}) {
	Logger.Printf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	// Could use a different prefix or output for errors
	Logger.Printf("ERROR: "+format, v...)
}
