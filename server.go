package hypertrace

import (
	"context"
	"fmt"
	mux "github.com/hyperjumptech/hyper-mux"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	hmux = mux.NewHyperMux()
)

func initRoutes() {
	hmux.AddRoute("/getHandshakePin", mux.MethodGet, getHandshakePin)
	hmux.AddRoute("/getTempIDs", mux.MethodGet, getTempIDs)
	hmux.AddRoute("/getUploadToken", mux.MethodGet, getUploadToken)
	hmux.AddRoute("/uploadData", mux.MethodPost, uploadData)
	hmux.AddRoute("/getTracing", mux.MethodGet, getTracing)
	hmux.AddRoute("/purgeTracing", mux.MethodGet, purgeTracing)
}

func StartServer() {
	initRoutes()

	var wait time.Duration

	// StartUpTime records first ime up
	startUpTime := time.Now()

	theServer := &http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           hmux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	go func() {
		err := theServer.ListenAndServe()
		if err != nil {
			logrus.Error(err.Error())
		}
	}()

	gracefulStop := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(gracefulStop, os.Interrupt)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	// Block until we receive our signal.
	<-gracefulStop

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	theServer.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	logrus.Info("shutting down........ bye")

	t := time.Now()
	upTime := t.Sub(startUpTime)
	fmt.Println("server was up for : ", upTime.String(), " *******")
	os.Exit(0)
}