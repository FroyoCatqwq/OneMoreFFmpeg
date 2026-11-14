package model_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func TestDryRunInstallOmitsJournalID(t *testing.T) {
	// env-set / uninstall already use omitempty; install must be consistent so
	// callers never see journal_id: "" and never treat it as a real transaction.
	payload, err := json.Marshal(model.InstallData{
		Schema: model.InstallSchema,
		DryRun: true,
	})
	require.NoError(t, err)
	require.False(t, strings.Contains(string(payload), "journal_id"),
		"dry-run must omit the field entirely: %s", payload)

	withID, err := json.Marshal(model.InstallData{
		Schema: model.InstallSchema, JournalID: "journal-0123456789abcdef",
	})
	require.NoError(t, err)
	require.Contains(t, string(withID), `"journal_id":"journal-0123456789abcdef"`)
}
