package env

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildEnvPlanSystemScope(t *testing.T) {
	opts := ApplyOptions{DataRoot: t.TempDir(), System: true}
	plan, err := buildEnvPlan(opts, true)
	require.NoError(t, err)
	require.Equal(t, "system", plan.details["scope"])

	if runtime.GOOS != "windows" {
		require.Equal(t, "/usr/local/bin", plan.details["shim_dir"])
		require.Equal(t, "/etc/profile.d/onemoreffmpeg.sh", plan.details["shell_rc_path"])
	}
}

func TestBuildEnvPlanUserScope(t *testing.T) {
	opts := ApplyOptions{DataRoot: t.TempDir()}
	plan, err := buildEnvPlan(opts, true)
	require.NoError(t, err)
	require.Equal(t, "user", plan.details["scope"])

	if runtime.GOOS != "windows" {
		require.NotEqual(t, "/usr/local/bin", plan.details["shim_dir"])
		require.NotEqual(t, "/etc/profile.d/onemoreffmpeg.sh", plan.details["shell_rc_path"])
		require.NotEmpty(t, plan.details["shell_rc_path"])
	}
}