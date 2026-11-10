package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/buildinfo"
	appcli "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/cli"
)

var (
	version   = "0.1.0"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	info := buildinfo.New(version, commit, buildTime)
	os.Exit(appcli.Run(ctx, appcli.RunOptions{
		Args:    os.Args,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: info,
	}))
}
