package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

func TestDoctorDefaultUnhealthyExitsZero(t *testing.T) {
	phase4Home(t)
	code, object, out := runJSON(t, "doctor", "--json")
	require.Equal(t, 0, code, out)
	data := object["data"].(map[string]any)
	require.Equal(t, false, data["healthy"])
}

func TestDoctorStrictUnhealthyExitsNonZero(t *testing.T) {
	phase4Home(t)
	code, object, out := runJSON(t, "doctor", "--strict", "--json")
	require.Equal(t, 1, code, out)
	require.Equal(t, "OMF-E6006", object["code"])
	details := object["details"].(map[string]any)
	report := details["report"].(map[string]any)
	require.Equal(t, false, report["healthy"])
}

func TestEnvSystemDryRunWithConsent(t *testing.T) {
	phase4Home(t)
	code, object, out := runJSON(t, "env", "set", "--system", "--authorize-system", "--json")
	require.Equal(t, 0, code, out)
	data := object["data"].(map[string]any)
	require.Equal(t, true, data["dry_run"])
	require.Equal(t, false, data["applied"])
}

func TestEnvSystemRefusedWithoutConsent(t *testing.T) {
	if platform.IsElevated() {
		t.Skip("process is elevated (root/Administrator); system scope is pre-authorized")
	}
	phase4Home(t)
	code, object, out := runJSON(t, "env", "set", "--system", "--json")
	require.Equal(t, 2, code, out)
	require.Equal(t, "OMF-E6003", object["code"])
}