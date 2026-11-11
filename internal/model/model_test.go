package model_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func TestDetectDataNilSlicesMarshalAsArrays(t *testing.T) {
	raw, err := json.Marshal(model.DetectData{Schema: model.DetectSchema})
	require.NoError(t, err)
	text := string(raw)
	require.Contains(t, text, `"installations":[]`)
	require.Contains(t, text, `"errors":[]`)
}

func TestInstallationNilSlicesMarshalAsArrays(t *testing.T) {
	raw, err := json.Marshal(model.Installation{ID: "inst"})
	require.NoError(t, err)
	text := string(raw)
	require.Contains(t, text, `"sources":[]`)
	require.Contains(t, text, `"warnings":[]`)
}

func TestValidateDataNilSlicesMarshalAsArrays(t *testing.T) {
	raw, err := json.Marshal(model.ValidateData{Schema: model.ValidateSchema})
	require.NoError(t, err)
	text := string(raw)
	require.Contains(t, text, `"checks":[]`)
	require.Contains(t, text, `"warnings":[]`)
}
