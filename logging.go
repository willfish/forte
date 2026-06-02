package main

import (
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/willfish/forte/internal/logx"
)

var forteApp *application.App

func initLogging(level string) {
	if err := logx.Configure(level); err != nil {
		// Last resort: stderr only via slog default.
		slog.Error("configure logging", "err", err)
	}
}

func applyLogging(level string) {
	if err := logx.Configure(level); err != nil {
		slog.Error("reconfigure logging", "err", err, "level", level)
		return
	}
	if forteApp != nil {
		forteApp.Logger = logx.Logger()
	}
}