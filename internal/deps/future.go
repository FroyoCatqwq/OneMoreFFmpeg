//go:build deps

package deps

import (
	_ "github.com/klauspost/compress/zstd"
	_ "github.com/sethvargo/go-retry"
	_ "github.com/ulikunitz/xz"
	_ "golang.org/x/sync/errgroup"
	_ "golang.org/x/sys/cpu"
	_ "golang.org/x/term"
	_ "golang.org/x/time/rate"
)
