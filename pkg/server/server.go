package server

import (
	"context"
	"github-actions-exporter/pkg/client"
	"github-actions-exporter/pkg/server/processor"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/xerrors"
)

func Run(a *Args) error {
	i := NewInstance()
	logger := client.NewStandardLogger(a.Verbose)
	i.SetLogger(logger)

	api, err := processor.NewAPI(processor.APISettings{
		Address:              a.APIAddress,
		MaxConnections:       a.APIMaxConnections,
		ReUsePort:            a.ReUsePort,
		KeepAlived:           a.KeepAlived,
		TCPKeepAliveInterval: time.Duration(a.TCPKeepAliveInterval) * time.Second,
		Logger:               i.Logger(),
	})
	if err != nil {
		return xerrors.Errorf("failed to create api: %w", err)
	}
	i.AddProcessor(api)

	monitor, err := processor.NewMonitor(processor.MonitorSettings{
		Address:               a.MonitorAddress,
		MaxConnections:        a.MonitorMaxConnections,
		JaegerEndpoint:        a.MonitoringJaegerEndpoint,
		EnableProfiling:       a.EnableProfiling,
		EnableTracing:         a.EnableTracing,
		TracingSampleRate:     a.TracingSampleRate,
		ReUsePort:             a.ReUsePort,
		KeepAlived:            a.KeepAlived,
		TCPKeepAliveInterval:  time.Duration(a.TCPKeepAliveInterval) * time.Second,
		CollectorLoopInterval: time.Duration(a.CollectorLoopInterval) * time.Second,
		HTTPClient:            i.HTTPClient(),
		Logger:                i.Logger(),
		Repository:            a.Repository,
		Token:                 a.Token,
	})
	if err != nil {
		return xerrors.Errorf("failed to create monitor: %w", err)
	}
	i.AddProcessor(monitor)

	i.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM)
	<-quit
	i.logger.Infof("Attempt to shutdown instance...\n")

	i.Shutdown(context.Background())
	return nil
}
