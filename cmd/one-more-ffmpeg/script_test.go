package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/buildinfo"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/cli"
)

// TestMain registers the `omf` command for txtar-based CLI scripts. Each
// script runs in an isolated work directory, exercising the real CLI code path
// through cli.Run (the same entry point as main).
func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"omf": func() {
			// testscript isolates HOME/USERPROFILE but not LOCALAPPDATA, which
			// paths.DataRoot prefers on Windows. Redirect it into the script's
			// work directory so scripts never read/write the host's registry.
			if work := os.Getenv("WORK"); work != "" {
				_ = os.Setenv("LOCALAPPDATA", filepath.Join(work, "appdata"))
			}
			code := cli.Run(context.Background(), cli.RunOptions{
				Args:   os.Args, // includes argv[0] ("omf"), matching main.go
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Version: buildinfo.Info{
					Version: "test", GoVersion: "go1.27", OS: "test", Arch: "test",
				},
			})
			os.Exit(code)
		},
	})
}

func TestCLIScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{Dir: "testdata"})
}
