// Package buildinfo exposes build metadata for the CLI version command.
package buildinfo

import (
	"runtime"
	"runtime/debug"
	"time"
)

type Info struct {
	Version   string
	Commit    string
	BuildTime string
	GoVersion string
	OS        string
	Arch      string
}

func New(version, commit, buildTime string) Info {
	if version == "" {
		version = "dev"
	}
	if commit == "" || commit == "unknown" || buildTime == "" || buildTime == "unknown" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				switch {
				case setting.Key == "vcs.revision" && setting.Value != "" && (commit == "" || commit == "unknown"):
					commit = setting.Value
				case setting.Key == "vcs.time" && setting.Value != "" && (buildTime == "" || buildTime == "unknown"):
					buildTime = setting.Value
				}
			}
		}
	}
	if commit == "" {
		commit = "unknown"
	}
	if !isRFC3339(buildTime) {
		buildTime = "unknown"
	}
	return Info{
		Version:   version,
		Commit:    commit,
		BuildTime: buildTime,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

func isRFC3339(value string) bool {
	if value == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}
