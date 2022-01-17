package main

import (
	"fmt"
	"github.com/hyperjumptech/hypertrace"
	"github.com/sirupsen/logrus"
	"time"
)

func main() {
	splash := `██   ██ ██    ██ ██████  ███████ ██████  ████████ ██████   █████   ██████ ███████ 
██   ██  ██  ██  ██   ██ ██      ██   ██    ██    ██   ██ ██   ██ ██      ██      
███████   ████   ██████  █████   ██████     ██    ██████  ███████ ██      █████   
██   ██    ██    ██      ██      ██   ██    ██    ██   ██ ██   ██ ██      ██      
██   ██    ██    ██      ███████ ██   ██    ██    ██   ██ ██   ██  ██████ ███████ 
                                                                                  
                                                                                  `
	fmt.Println(splash)
	levels := map[string]logrus.Level{
		"TRACE": logrus.TraceLevel,
		"DEBUG": logrus.DebugLevel,
		"INFO":  logrus.InfoLevel,
		"WARN":  logrus.WarnLevel,
		"ERROR": logrus.ErrorLevel,
		"FATAL": logrus.FatalLevel,
	}
	if lvl, ok := levels[hypertrace.ConfigGet("loglevel")]; ok {
		logrus.SetLevel(lvl)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	logrus.Infof("HYPERTRACE - STARTED : %v", time.Now().Format(time.RFC850))
	hypertrace.StartServer()
}
