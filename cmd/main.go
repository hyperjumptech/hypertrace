package main

import (
	"github.com/hyperjumptech/hypertrace"
	"github.com/sirupsen/logrus"
	"time"
)

func main() {
	logrus.Infof("HYPERTRACE - STARTED : %v", time.Now())
	hypertrace.StartServer()
}
