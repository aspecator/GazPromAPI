package logMachine

import (
	"log"
	"os"
)

type logMachine struct {
	info *log.Logger
	err  *log.Logger
	File *os.File
	v    int
}

func New() *logMachine {
	logMachine := &logMachine{}
	logMachine.info = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	logMachine.err = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime)
	logMachine.File = nil
	logMachine.v = 0
	return logMachine
}

var std = New()

func Default() *logMachine { return std }

func (l *logMachine) SetLogFile(logFileName string) {
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}

	l.File = logFile
	l.info.SetOutput(logFile)
	l.err.SetOutput(logFile)
}

func (l *logMachine) SetVerbLevel(v int) {
	l.v = v
}

func (l *logMachine) Info(v ...any) {
	if l.v >= 1 {
		l.info.Print(v...)
	}
}

func (l *logMachine) Error(v ...any) {
	l.err.Print(v...)
}

func (l *logMachine) Fatal(v ...any) {
	l.err.Fatal(v...)
}

//----------------------------

func SetLogFile(logFileName string) {
	std.SetLogFile(logFileName)
}

func SetVerbLevel(v int) {
	std.SetVerbLevel(v)
}

func Info(v ...any) {
	std.Info(v...)
}

func Error(v ...any) {
	std.Error(v...)
}

func Fatal(v ...any) {
	std.Fatal(v...)
}
