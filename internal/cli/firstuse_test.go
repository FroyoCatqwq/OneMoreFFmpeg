package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOnlyApplyCommandsMayPersistTheFirstUseMarker(t *testing.T) {
	// A read-only command or a dry-run must not create the data root or the
	// marker; only an invocation that is allowed to write may leave a trace.
	require.True(t, writesSomething([]string{"install", "--apply"}))
	require.True(t, writesSomething([]string{"env", "set", "--apply"}))
	require.True(t, writesSomething([]string{"--json", "uninstall", "--apply"}))

	require.False(t, writesSomething([]string{"version"}))
	require.False(t, writesSomething([]string{"detect", "--json"}))
	require.False(t, writesSomething([]string{"install"}))
	require.False(t, writesSomething([]string{"doctor", "--strict"}))
	require.False(t, writesSomething(nil))

	// download always writes; capabilities writes its cache unless --no-cache.
	require.True(t, writesSomething([]string{"download", "--release", "r.json"}))
	require.True(t, writesSomething([]string{"capabilities", "--dir", "x"}))
	require.False(t, writesSomething([]string{"capabilities", "--dir", "x", "--no-cache"}))
}
