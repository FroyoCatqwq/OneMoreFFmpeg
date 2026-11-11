package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	urfave "github.com/urfave/cli/v3"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/contracts"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func wantsJSON(args []string) bool {
	value := false
	needsValue := map[string]bool{
		"--config":          true,
		"-c":                true,
		"--log-level":       true,
		"--dir":             true,
		"--roots":           true,
		"--max-depth":       true,
		"--max-files":       true,
		"--timeout":         true,
		"--probe-timeout":   true,
		"--installation-id": true,
		"--help-topic":      true,
		"--cache-dir":       true,
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if needsValue[arg] {
			i++
			continue
		}
		switch {
		case arg == "--json" || arg == "-json":
			value = true
		case strings.HasPrefix(arg, "--json=") || strings.HasPrefix(arg, "-json="):
			raw := arg[strings.IndexByte(arg, '=')+1:]
			parsed, err := strconv.ParseBool(raw)
			if err == nil {
				value = parsed
			}
		}
	}
	return value
}

func jsonEnabled(cmd *urfave.Command) bool {
	if cmd == nil {
		return false
	}
	if cmd.Bool("json") {
		return true
	}
	root := cmd.Root()
	if root != nil && root != cmd && root.Bool("json") {
		return true
	}
	return false
}

func writerFor(cmd *urfave.Command) io.Writer {
	if cmd != nil && cmd.Writer != nil {
		return cmd.Writer
	}
	if cmd != nil && cmd.Root() != nil && cmd.Root().Writer != nil {
		return cmd.Root().Writer
	}
	return os.Stdout
}

func encodeJSON(w io.Writer, value any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return apperrors.New(apperrors.E7006, "stdout JSON serialization failed").WithCause(err.Error())
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return apperrors.New(apperrors.E7006, "stdout JSON write failed").WithCause(err.Error())
	}
	return nil
}

func writeResult(cmd *urfave.Command, command string, data any, human string) error {
	if err := contracts.ValidateData(data); err != nil {
		return err
	}
	if !jsonEnabled(cmd) {
		_, err := fmt.Fprint(writerFor(cmd), human)
		return err
	}
	return encodeJSON(writerFor(cmd), model.Result{
		Schema:  model.ResultSchema,
		OK:      true,
		Command: command,
		Data:    data,
	})
}

func errorObject(err error) model.ErrorObject {
	typed := apperrors.From(err)
	return model.ErrorObject{
		Schema:  model.ErrorSchema,
		OK:      false,
		Code:    string(typed.Code),
		Message: typed.Message,
		Details: typed.Details,
		Cause:   typed.Cause,
	}
}

func writeFailure(stdout, stderr io.Writer, asJSON bool, err error) {
	obj := errorObject(err)
	if asJSON {
		_ = encodeJSON(stdout, obj)
		return
	}
	if obj.Cause != "" {
		_, _ = fmt.Fprintf(stderr, "error: %s: %s (cause: %s)\n", obj.Code, obj.Message, obj.Cause)
		return
	}
	_, _ = fmt.Fprintf(stderr, "error: %s: %s\n", obj.Code, obj.Message)
}
