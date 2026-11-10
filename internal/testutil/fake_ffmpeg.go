// Package testutil builds cross-platform fake FFmpeg executables for tests.
package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type FakeFFmpeg struct {
	Dir         string
	FFmpegPath  string
	FFprobePath string
	FFplayPath  string
}

const fakeSource = `package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	if s := os.Getenv("OMF_FAKE_GRANDCHILD_MS"); s != "" && os.Getenv("OMF_FAKE_CHILD") == "" {
		child := exec.Command(os.Args[0], os.Args[1:]...)
		child.Env = append(os.Environ(),
			"OMF_FAKE_CHILD=1",
			"OMF_FAKE_SLEEP_MS="+s,
			"OMF_FAKE_GRANDCHILD_MS=",
		)
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		_ = child.Start()
		os.Exit(0)
	}
	if s := os.Getenv("OMF_FAKE_STDOUT"); s != "" {
		fmt.Print(s)
	}
	if s := os.Getenv("OMF_FAKE_STDERR"); s != "" {
		fmt.Fprint(os.Stderr, s)
	}
	if s := os.Getenv("OMF_FAKE_BYTES"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			fmt.Print(strings.Repeat("x", n))
		}
	}
	if s := os.Getenv("OMF_FAKE_ENV"); s != "" {
		fmt.Print(os.Getenv(s))
	}
	if s := os.Getenv("OMF_FAKE_SLEEP_MS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			time.Sleep(time.Duration(n) * time.Millisecond)
		}
	}
	if s := os.Getenv("OMF_FAKE_EXIT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			os.Exit(n)
		}
	}
}
`

func NewFakeFFmpeg(t testing.TB) *FakeFFmpeg {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not found in PATH")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "fake_main.go")
	if err := os.WriteFile(src, []byte(fakeSource), 0o644); err != nil {
		t.Fatalf("write fake source: %v", err)
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	buildPath := filepath.Join(dir, "fake-ffmpeg"+ext)
	build := exec.Command("go", "build", "-o", buildPath, src)
	build.Dir = dir
	build.Env = buildEnv()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake ffmpeg: %v\n%s", err, string(out))
	}
	data, err := os.ReadFile(buildPath)
	if err != nil {
		t.Fatalf("read fake binary: %v", err)
	}
	paths := make([]string, 3)
	for i, name := range []string{"ffmpeg", "ffprobe", "ffplay"} {
		path := filepath.Join(dir, name+ext)
		if err := os.WriteFile(path, data, 0o755); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		paths[i] = path
	}
	return &FakeFFmpeg{
		Dir:         dir,
		FFmpegPath:  paths[0],
		FFprobePath: paths[1],
		FFplayPath:  paths[2],
	}
}

func buildEnv() []string {
	env := os.Environ()
	out := make([]string, 0, len(env)+4)
	for _, pair := range env {
		key, _, _ := strings.Cut(pair, "=")
		switch strings.ToUpper(key) {
		case "GOOS", "GOARCH", "GOFLAGS", "CGO_ENABLED":
			continue
		}
		out = append(out, pair)
	}
	out = append(out,
		"GOOS="+runtime.GOOS,
		"GOARCH="+runtime.GOARCH,
		"CGO_ENABLED=0",
		"GO111MODULE=off",
	)
	return out
}

func (f *FakeFFmpeg) Env(stdout, stderr string, exitCode int) []string {
	return []string{
		"OMF_FAKE_STDOUT=" + stdout,
		"OMF_FAKE_STDERR=" + stderr,
		fmt.Sprintf("OMF_FAKE_EXIT=%d", exitCode),
	}
}
