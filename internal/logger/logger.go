package logger

import (
	"log"
	"os"
)

var loggerLevel = InfoLevel
var logger = log.New(os.Stderr, "", log.LstdFlags)

const (
	_ = iota
	InfoLevel
	DebugLevel
	WarnLevel
	ErrorLevel
	ExitLevel
)

func Info(format string, args ...interface{}) {
	if loggerLevel > InfoLevel {
		return
	}
	logger.Printf("[info]"+format, args...)
}
func Debug(format string, args ...interface{}) {
	if loggerLevel > DebugLevel {
		return
	}
	logger.Printf("[debug]"+format, args...)
}
func Warn(format string, args ...interface{}) {
	if loggerLevel > WarnLevel {
		return
	}
	logger.Printf("[warn]"+format, args...)
}
func Error(format string, args ...interface{}) {
	if loggerLevel > ErrorLevel {
		return
	}
	logger.Printf("[error]"+format, args...)
}
func Exit(format string, args ...interface{}) {
	if loggerLevel > ExitLevel {
		return
	}
	logger.Printf("[exit]"+format, args...)
	os.Exit(1)
}
func SetLogLevel(n int) {
	loggerLevel = n
}
