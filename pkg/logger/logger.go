package logger

import (
	"fmt"
	"log"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarningLevel
	ErrorLevel
)

type Domain string

const (
	Application Domain = "application"
	Network     Domain = "network"
	Election    Domain = "election"
)

var (
	currentLevel   = InfoLevel
	enabledDomains = map[Domain]bool{
		Application: true,
		Network:     true,
		Election:    true,
	}
)

func init() {
	// TODO: implement init with env variables etc.
}

func logf(level Level, domain Domain, format string, v ...interface{}) {
	if level >= currentLevel && enabledDomains[domain] {
		levelStr := "INFO"
		switch level {
		case DebugLevel:
			levelStr = "DEBUG"
		case WarningLevel:
			levelStr = "WARN"
		case ErrorLevel:
			levelStr = "ERROR"
		}
		msg := fmt.Sprintf(format, v...)
		log.Printf("[%s][%s] %s", levelStr, domain, msg)
	}
}

func Debugf(domain Domain, format string, v ...interface{}) {
	logf(DebugLevel, domain, format, v...)
}

func Infof(domain Domain, format string, v ...interface{}) {
	logf(InfoLevel, domain, format, v...)
}

func Warningf(domain Domain, format string, v ...interface{}) {
	logf(WarningLevel, domain, format, v...)
}

func Errorf(domain Domain, format string, v ...interface{}) {
	logf(ErrorLevel, domain, format, v...)
}

func Fatalf(domain Domain, format string, v ...interface{}) {
	logf(ErrorLevel, domain, format, v...)
	log.Fatalf(format, v...)
}
