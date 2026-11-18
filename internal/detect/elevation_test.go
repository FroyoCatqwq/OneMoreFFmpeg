package detect

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func elevatedDetector(elevated bool) *Detector {
	d := New()
	d.IsElevated = func() bool { return elevated }
	return d
}

func TestFullScanRequiresElevation(t *testing.T) {
	opts := DefaultOptions()
	opts.Scan = true // no explicit Roots -> full scan
	_, err := elevatedDetector(false).Detect(context.Background(), opts)
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E6003, typed.Code)
	require.Equal(t, false, typed.Details["elevated"])
}

func TestFullScanAllowedWhenElevated(t *testing.T) {
	opts := DefaultOptions()
	opts.Scan = true
	opts.MaxFiles = 50
	opts.MaxDepth = 1
	opts.NoPath = true
	opts.NoEnv = true
	opts.NoCommon = true
	opts.NoValidate = true
	// With elevation the scan proceeds (may find nothing; must not be E6003).
	_, err := elevatedDetector(true).Detect(context.Background(), opts)
	if err != nil {
		typed, ok := apperrors.As(err)
		if ok {
			require.NotEqual(t, apperrors.E6003, typed.Code, "elevated scan must not be blocked: %v", err)
		}
	}
}

func TestScopedScanDoesNotRequireElevation(t *testing.T) {
	opts := DefaultOptions()
	opts.Scan = true
	opts.Roots = []string{t.TempDir()} // explicit scoped root
	opts.MaxDepth = 1
	opts.NoPath = true
	opts.NoEnv = true
	opts.NoCommon = true
	opts.NoValidate = true
	_, err := elevatedDetector(false).Detect(context.Background(), opts)
	require.NoError(t, err, "explicit roots are a scoped scan, no elevation required")
}

func TestRecursiveDirAtFilesystemRootRequiresElevation(t *testing.T) {
	root := "/"
	if runtime.GOOS == "windows" {
		root = `C:\`
	}
	opts := DefaultOptions()
	opts.Dir = root
	opts.Recursive = true
	_, err := elevatedDetector(false).Detect(context.Background(), opts)
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E6003, typed.Code)
}

func TestIsFilesystemRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		require.True(t, isFilesystemRoot(`C:\`))
		require.False(t, isFilesystemRoot(`C:\Users`))
	} else {
		require.True(t, isFilesystemRoot("/"))
		require.False(t, isFilesystemRoot("/home"))
	}
}
