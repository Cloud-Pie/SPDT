package util

import (
	"log"
	"io"
	"os"
	"path/filepath"
)

type Logger struct {
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
}

func NewLogger() *Logger {
	Log := new (Logger)
	Log.SetLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	return  Log
}


func (Log *Logger) SetLogger(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Log.Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Log.Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Log.Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Log.Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

func (Log *Logger) SetLogFile(logFile string)  error {
	if logFile == "" {
		logFile = DEFAULT_LOGFILE
	}
	os.MkdirAll(filepath.Dir(logFile), 0700)
	file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		Log.Error.Println("Cannot access log file.")
	} else {
		multiOutput := io.MultiWriter(file, os.Stdout)
		multiError := io.MultiWriter(file, os.Stderr)
		Log.SetLogger(multiOutput, multiOutput, multiOutput, multiError)
	}
	return err
}