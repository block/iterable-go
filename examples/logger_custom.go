package main

import (
	"fmt"
	"log"
	"os"
	"time"

	iterable_go "iterable-go"
	"iterable-go/logger"
)

func logger_custom(apiKey string) {
	// Create our custom logger
	myLogger := NewMyAwesomeLogger("IterableClient")

	fmt.Println("=== Custom Logger Demo ===")
	myLogger.Debugf("This is a debug message with parameter: %s", "debug_value")
	myLogger.Infof("This is an info message with number: %d", 42)
	myLogger.Warnf("This is a warning message about %s", "something important")
	myLogger.Errorf("This is an error message with error code: %d", 500)
	fmt.Println()

	fmt.Println("=== Creating Iterable Client with Custom Logger ===")
	client := iterable_go.NewClient(
		apiKey,
		iterable_go.WithLogger(myLogger),
	)

	fmt.Println("=== Creating Batch Client with Custom Logger ===")
	_ = iterable_go.NewBatch(
		client,
		iterable_go.WithBatchLogger(myLogger),
		iterable_go.WithBatchFlushQueueSize(5),
		iterable_go.WithBatchFlushInterval(3*time.Second),
	)
}

// MyAwesomeLogger implements the logger.Logger interface
// with custom formatting and colors
type MyAwesomeLogger struct {
	prefix       string
	actualLogger *log.Logger
}

var _ logger.Logger = &MyAwesomeLogger{}

func NewMyAwesomeLogger(prefix string) *MyAwesomeLogger {
	return &MyAwesomeLogger{
		prefix: prefix,
		// No default prefix, we'll add our own
		actualLogger: log.New(os.Stdout, "", 0),
	}
}

// ANSI color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorGray   = "\033[37m"
)

func (l *MyAwesomeLogger) Debugf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.actualLogger.Printf(
		"%s[%s] [%s] [DEBUG] %s%s\n",
		ColorGray, l.timestamp(), l.prefix, message, ColorReset,
	)
}

func (l *MyAwesomeLogger) Infof(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.actualLogger.Printf(
		"%s[%s] [%s] [INFO]  %s%s\n",
		ColorBlue, l.timestamp(), l.prefix, message, ColorReset,
	)
}

func (l *MyAwesomeLogger) Warnf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.actualLogger.Printf(
		"%s[%s] [%s] [WARN]  %s%s\n",
		ColorYellow, l.timestamp(), l.prefix, message, ColorReset,
	)
}

func (l *MyAwesomeLogger) Errorf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.actualLogger.Printf(
		"%s[%s] [%s] [ERROR] %s%s\n",
		ColorRed, l.timestamp(), l.prefix, message, ColorReset,
	)
}

func (l *MyAwesomeLogger) timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
