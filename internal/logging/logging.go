package logging

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"runtime"
)

type aggregatedLogger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
}

var al *aggregatedLogger

func Init(out io.Writer) {
	flags := log.LstdFlags
	al = &aggregatedLogger{
		infoLogger:  log.New(out, "", flags),
		warnLogger:  log.New(out, "", flags),
		errorLogger: log.New(out, "", flags),
	}
}

func logWithCaller(logger *log.Logger, prefix string, v ...interface{}) {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		shortFile := filepath.Base(file)
		logger.Println(fmt.Sprintf("%s:%d %s", shortFile, line, prefix), fmt.Sprint(v...))
	} else {
		logger.Println(fmt.Sprint(v...))
	}
}

func logfWithCaller(logger *log.Logger, prefix, format string, v ...interface{}) {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		shortFile := filepath.Base(file)
		logger.Printf(fmt.Sprintf("%s:%d %s %s", shortFile, line, prefix, format), v...)
	} else {
		logger.Printf(format, v...)
	}
}

func Info(v ...interface{}) {
	logWithCaller(al.infoLogger, colorWhite("INFO:"), colorWhiteRange(v...))
}

func Warn(v ...interface{}) {
	logWithCaller(al.warnLogger, colorWhite("WARN:"), colorYellowRange(v...))
}

func Error(v ...interface{}) {
	logWithCaller(al.errorLogger, colorWhite("ERROR:"), colorRedRange(v...))
}

func Infof(format string, v ...any) {
	logfWithCaller(al.infoLogger, colorWhite("INFO:"), colorWhite(format), v)
}

func Warnf(format string, v ...any) {
	logfWithCaller(al.warnLogger, colorWhite("WARN:"), colorYellow(format), v)
}

func Errorf(format string, v ...any) {
	logfWithCaller(al.errorLogger, colorWhite("ERROR:"), colorRed(format), v)
}
