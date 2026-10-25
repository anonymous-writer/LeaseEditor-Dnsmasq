package app

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

type appLogLevel int

const (
	levelDebug appLogLevel = iota
	levelInfo
	levelWarn
	levelError
)

var (
	appLogMu         sync.RWMutex
	appLogLevelValue = parseAppLogLevel(os.Getenv("LOG_LEVEL"))
	appLogger        = log.New(os.Stdout, "", log.LstdFlags)
)

// parseAppLogLevel reads the minimum severity. Empty or unknown values default to INFO.
func parseAppLogLevel(value string) appLogLevel {
	switch strings.ToUpper(strings.Trim(strings.TrimSpace(value), "\"'")) {
	case "DEBUG":
		return levelDebug
	case "WARN", "WARNING":
		return levelWarn
	case "ERROR":
		return levelError
	default:
		return levelInfo
	}
}

func validLogLevel(value string) bool {
	switch strings.ToUpper(strings.Trim(strings.TrimSpace(value), "\"'")) {
	case "DEBUG", "INFO", "WARN", "ERROR":
		return true
	default:
		return false
	}
}

// SetLogLevel changes the minimum severity at runtime, primarily for tests.
func SetLogLevel(value string) {
	appLogMu.Lock()
	appLogLevelValue = parseAppLogLevel(value)
	appLogMu.Unlock()
}

func appLogf(level appLogLevel, label, format string, args ...any) {
	appLogMu.RLock()
	enabled := level >= appLogLevelValue
	appLogMu.RUnlock()
	if enabled {
		appLogger.Printf("[%s] %s", label, fmt.Sprintf(format, args...))
	}
}

func LogDebugf(format string, args ...any) { appLogf(levelDebug, "DEBUG", format, args...) }
func LogInfof(format string, args ...any)  { appLogf(levelInfo, "INFO", format, args...) }
func LogWarnf(format string, args ...any)  { appLogf(levelWarn, "WARN", format, args...) }
func LogErrorf(format string, args ...any) { appLogf(levelError, "ERROR", format, args...) }

// LogFatalf records a fatal startup error and exits with status code 1.
func LogFatalf(format string, args ...any) {
	appLogf(levelError, "FATAL", format, args...)
	os.Exit(1)
}
