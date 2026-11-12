package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/buildinfo"
	appcli "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/cli"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	info := buildinfo.New(version, commit, buildTime)
	code := appcli.Run(ctx, appcli.RunOptions{
		Args:    os.Args,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: info,
	})
	stop()
	os.Exit(code)
}
