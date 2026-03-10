package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var currentLevel Level

func Init() {
	levelStr := strings.ToUpper(os.Getenv("CARELINK_LOG_LEVEL"))
	switch levelStr {
	case "DEBUG":
		currentLevel = DEBUG
	case "INFO":
		currentLevel = INFO
	case "WARN":
		currentLevel = WARN
	case "ERROR":
		currentLevel = ERROR
	default:
		currentLevel = INFO
	}
}

func Debug(component, format string, v ...interface{}) {
	if currentLevel <= DEBUG {
		log.Printf("[DEBUG] [%s] %s", component, fmt.Sprintf(format, v...))
	}
}

func Info(component, format string, v ...interface{}) {
	if currentLevel <= INFO {
		log.Printf("[INFO] [%s] %s", component, fmt.Sprintf(format, v...))
	}
}

func Warn(component, format string, v ...interface{}) {
	if currentLevel <= WARN {
		log.Printf("[WARN] [%s] %s", component, fmt.Sprintf(format, v...))
	}
}

func Error(component, format string, v ...interface{}) {
	if currentLevel <= ERROR {
		log.Printf("[ERROR] [%s] %s", component, fmt.Sprintf(format, v...))
	}
}

func Fatal(component, format string, v ...interface{}) {
	log.Fatalf("[FATAL] [%s] %s", component, fmt.Sprintf(format, v...))
}
