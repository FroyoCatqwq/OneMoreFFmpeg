package doctor_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/doctor"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
)

func TestDoctorReportsUnhealthyWithoutCurrent(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, journal.Write(filepath.Join(root, "journal"), journal.Record{
		ID: "journal-pending", Action: "install", Status: journal.StatusPending,
	}))

	report, err := doctor.Run(context.Background(), doctor.Options{DataRoot: root, Env: []string{"PATH=/usr/bin"}})
	require.NoError(t, err)
	require.False(t, report.Healthy)
	require.Len(t, report.Checks, 5)
	require.Equal(t, []string{"journal-pending"}, report.JournalPending)

	byName := map[string]bool{}
	for _, check := range report.Checks {
		byName[check.Name] = check.OK
	}
	require.False(t, byName["current_exists"])
	require.False(t, byName["trio_valid"])
	require.False(t, byName["no_pending_journal"])
}
