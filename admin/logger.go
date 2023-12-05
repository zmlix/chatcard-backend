package admin

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var infoLogger *log.Logger 
var warnLogger *log.Logger
var errorLogger *log.Logger
var debugLogger *log.Logger


func init() {
	logDir := "logs"
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Println("Error loading location:", err)
		return
	}
	logFilePath := filepath.Join(logDir, fmt.Sprintf("share-%s.log", time.Now().In(loc).Format("2006-01-02")))
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.MkdirAll(logDir, 0755)
	}	
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0755)
	if err != nil {
		log.Fatalln("fail to create share.log", err)
	}
	infoLogger = log.New(logFile, "[INFO]", log.LstdFlags|log.Lshortfile)
	warnLogger = log.New(logFile, "[WARN]", log.LstdFlags|log.Lshortfile)
	errorLogger = log.New(logFile, "[ERROR]", log.LstdFlags|log.Lshortfile)
	debugLogger = log.New(logFile, "[DEBUG]", log.LstdFlags|log.Lshortfile)
}

func Info(format string, v ...interface{}) {
	if v == nil{
		infoLogger.Println(format)
	} else {
		infoLogger.Printf(format+"\t", v)
	}
}

func Warn(format string, v ...interface{}) {
	if v == nil{
		warnLogger.Println(format)
	} else {
		warnLogger.Printf(format+"\t", v)
	}
}

func Error(format string, v ...interface{}) {
	if v == nil{
		errorLogger.Println(format)
	} else {
		errorLogger.Printf(format+"\t", v)
	}
}

func Debug(format string, v ...interface{}) {
	if v == nil{
		debugLogger.Println(format)
	} else {
		debugLogger.Printf(format+"\t", v)
	}
}

func Panic(v ...interface{}) {
	log.Panic(v...)
}

func Fatal(v ...interface{}) {
	log.Fatal(v...)
}
