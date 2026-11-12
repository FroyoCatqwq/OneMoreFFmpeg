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
	"path/filepath"
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
	name := strings.ToLower(filepath.Base(os.Args[0]))
	if ext := filepath.Ext(name); ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	if v := os.Getenv("OMF_FAKE_VERSION_" + strings.ToUpper(name)); v != "" {
		fmt.Printf("%s version %s\n", name, v)
	}

	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}
	argEnv := ""
	switch arg {
	case "-L", "-license":
		argEnv = "OMF_FAKE_STDOUT_LICENSE"
	case "-version":
		argEnv = "OMF_FAKE_STDOUT_VERSION"
	case "-h":
		argEnv = "OMF_FAKE_STDOUT_HELP"
	case "-muxers":
		argEnv = "OMF_FAKE_STDOUT_MUXERS"
	case "-demuxers":
		argEnv = "OMF_FAKE_STDOUT_DEMUXERS"
	case "-devices":
		argEnv = "OMF_FAKE_STDOUT_DEVICES"
	case "-decoders":
		argEnv = "OMF_FAKE_STDOUT_DECODERS"
	case "-encoders":
		argEnv = "OMF_FAKE_STDOUT_ENCODERS"
	case "-filters":
		argEnv = "OMF_FAKE_STDOUT_FILTERS"
	case "-pix_fmts":
		argEnv = "OMF_FAKE_STDOUT_PIX_FMTS"
	case "-layouts":
		argEnv = "OMF_FAKE_STDOUT_LAYOUTS"
	case "-sample_fmts":
		argEnv = "OMF_FAKE_STDOUT_SAMPLE_FMTS"
	}
	if argEnv != "" {
		if s := os.Getenv(argEnv); s != "" {
			fmt.Print(s)
		}
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
	if s := os.Getenv("OMF_FAKE_FAIL_ARGS"); s != "" {
		for _, candidate := range strings.Split(s, ",") {
			if strings.TrimSpace(candidate) == arg {
				os.Exit(3)
			}
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
	return NewFakeFFmpegAt(t, "")
}

// NewFakeFFmpegAt places the fake trio in dir (created when needed). An empty
// dir uses a fresh temporary directory.
func NewFakeFFmpegAt(t testing.TB, dir string) *FakeFFmpeg {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not found in PATH")
	}
	if dir == "" {
		dir = t.TempDir()
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create fake dir: %v", err)
	}
	buildDir := t.TempDir()
	src := filepath.Join(buildDir, "fake_main.go")
	if err := os.WriteFile(src, []byte(fakeSource), 0o644); err != nil {
		t.Fatalf("write fake source: %v", err)
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	buildPath := filepath.Join(buildDir, "fake-ffmpeg"+ext)
	build := exec.Command("go", "build", "-o", buildPath, src)
	build.Dir = buildDir
	build.Env = buildEnv()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake ffmpeg: %v\n%s", err, string(out))
	}
	data, err := os.ReadFile(buildPath)
	if err != nil {
		t.Fatalf("read fake binary: %v", err)
	}
	paths := make([]string, 0, 3)
	for _, name := range []string{"ffmpeg", "ffprobe", "ffplay"} {
		path := filepath.Join(dir, name+ext)
		if err := os.WriteFile(path, data, 0o755); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		paths = append(paths, path)
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

func (f *FakeFFmpeg) VersionEnv(ffmpeg, ffprobe, ffplay string) []string {
	return append(f.Env("", "", 0),
		"OMF_FAKE_VERSION_FFMPEG="+ffmpeg,
		"OMF_FAKE_VERSION_FFPROBE="+ffprobe,
		"OMF_FAKE_VERSION_FFPLAY="+ffplay,
	)
}

func (f *FakeFFmpeg) CapabilityEnv() []string {
	return append(f.Env("", "", 0),
		"OMF_FAKE_VERSION_FFMPEG=",
		"OMF_FAKE_VERSION_FFPROBE=",
		"OMF_FAKE_VERSION_FFPLAY=",
		"OMF_FAKE_STDOUT_VERSION=ffmpeg version n9.0.2\nconfiguration: --enable-gpl --enable-version3\n",
		"OMF_FAKE_STDOUT_LICENSE=GPL version 3 or later\nThis program is free software.\n",
		"OMF_FAKE_STDOUT_HELP=Help: full\n",
		"OMF_FAKE_STDOUT_MUXERS=File formats:\n D. = Demuxing supported\n .E = Muxing supported\n --\n E mp4 MP4 (MPEG-4 Part 14)\n",
		"OMF_FAKE_STDOUT_DEMUXERS=File formats:\n D. = Demuxing supported\n .E = Muxing supported\n --\n D mov,mp4 QuickTime / MOV\n",
		"OMF_FAKE_STDOUT_DEVICES=Devices:\n D. = Demuxing supported\n .E = Muxing supported\n --\n D dshow DirectShow capture\n",
		"OMF_FAKE_STDOUT_DECODERS=Decoders:\n V..... = Video\n A..... = Audio\n ------\n V....D h264 H.264 / AVC\n A....D aac AAC (Advanced Audio Coding)\n",
		"OMF_FAKE_STDOUT_ENCODERS=Encoders:\n V..... = Video\n ------\n V....D libx264 H.264 encoder\n",
		"OMF_FAKE_STDOUT_FILTERS=Filters:\n T.. = Timeline support\n ... scale V->V Scale the input video\n",
		"OMF_FAKE_STDOUT_PIX_FMTS=Pixel formats:\n I.... = Supported Input\n .O... = Supported Output\n --\n IO... yuv420p planar YUV 4:2:0, 12bpp\n",
		"OMF_FAKE_STDOUT_LAYOUTS=Individual channels:\n NAME DESCRIPTION\n FL front left\n FR front right\n\nStandard channel layouts:\n NAME DECOMPOSITION\n stereo FL+FR\n",
		"OMF_FAKE_STDOUT_SAMPLE_FMTS=name depth\n u8 8\n s16 16\n",
	)
}
