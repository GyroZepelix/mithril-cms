package logging

import (
	"io"
	"log"
)

type aggregatedLogger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
}

var al *aggregatedLogger

func Init(out io.Writer) {
	flags := log.LstdFlags | log.Lshortfile | log.Lmsgprefix
	al = &aggregatedLogger{
		infoLogger:  log.New(out, colorWhite("INFO: "), flags),
		warnLogger:  log.New(out, colorYellow("WARN: "), flags),
		errorLogger: log.New(out, colorRed("ERROR: "), flags),
	}
}

func Info(v ...interface{}) {
	al.infoLogger.Println(colorWhiteRange(v...))
}

func Warn(v ...interface{}) {
	al.warnLogger.Println(colorYellowRange(v...))
}

func Error(v ...interface{}) {
	al.errorLogger.Println(colorRedRange(v...))
}

func Infof(format string, v ...any) {
	al.infoLogger.Printf(colorWhite(format), v...)
}

func Warnf(format string, v ...any) {
	al.warnLogger.Printf(colorYellow(format), v...)
}

func Errorf(format string, v ...any) {
	al.errorLogger.Printf(colorRed(format), v...)
}
