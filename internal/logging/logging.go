// Package logging wires structured JSON logging via Zap, matching the main
// app's convention (see auto-repair-shop's internal/pkg/logger). Lambda's
// runtime already timestamps/collects stdout into CloudWatch, so this just
// needs JSON output — no custom sink.
package logging

import "go.uber.org/zap"

func New() *zap.Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		// No logger yet to report this through; this only fails on a
		// misconfigured zap.Config, not a runtime condition.
		panic(err)
	}
	return logger
}
