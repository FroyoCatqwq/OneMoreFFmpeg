//go:build windows

package platform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJunctionSubstituteName(t *testing.T) {
	require.Equal(t, `\??\C:\ffmpeg`, junctionSubstituteName(`C:\ffmpeg`))
	// UNC targets must use the \??\UNC\ prefix, not \??\ directly.
	require.Equal(t, `\??\UNC\server\share\ffmpeg`, junctionSubstituteName(`\\server\share\ffmpeg`))
}