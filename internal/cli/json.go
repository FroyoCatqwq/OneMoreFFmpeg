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
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if takesValue[arg] {
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
		// Mark this specific error so writeFailure does not attempt a second
		// stdout write (which would mix a partial object with an error object).
		return apperrors.New(apperrors.E7006, "stdout JSON write failed").WithCause(err.Error()).
			WithDetails(map[string]any{"stdout_write_started": true})
	}
	return nil
}

func writeResult(cmd *urfave.Command, command string, data any, human string) error {
	// Validate the contracted (command, data.schema) pair before anything is
	// written so a regression can never leak an unparseable/off-contract payload.
	if err := contracts.ValidateCommandData(command, data); err != nil {
		return err
	}
	if err := contracts.ValidateDataFields(data); err != nil {
		return err
	}
	if !jsonEnabled(cmd) {
		if _, err := fmt.Fprint(writerFor(cmd), human); err != nil {
			return apperrors.New(apperrors.E7006, "stdout write failed").WithCause(err.Error())
		}
		return nil
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
	obj := model.ErrorObject{
		Schema:  model.ErrorSchema,
		OK:      false,
		Code:    string(typed.Code),
		Message: typed.Message,
		Details: typed.Details,
		Cause:   typed.Cause,
	}
	// Never emit an off-contract error envelope: an oversized or credential-
	// bearing message/cause must degrade to a fixed safe error rather than
	// leaking data or breaking consumers' parsers.
	if validationErr := contracts.ErrorObjectFieldsAreValid(obj.Code, obj.Message, obj.Details, obj.Cause); validationErr != nil {
		return model.ErrorObject{
			Schema:  model.ErrorSchema,
			OK:      false,
			Code:    string(apperrors.E9001),
			Message: "internal error envelope violated the contract",
			Details: map[string]any{},
		}
	}
	return obj
}

func writeFailure(stdout, stderr io.Writer, asJSON bool, err error) {
	obj := errorObject(err)
	// Only an E7006 that actually started writing to stdout must stay on
	// stderr: a second stdout write would mix a partial object with the error.
	// An E7006 raised before any write (e.g. field validation) still owes the
	// caller a parseable error envelope.
	if obj.Code == string(apperrors.E7006) && obj.Details != nil && obj.Details["stdout_write_started"] == true {
		_, _ = fmt.Fprintf(stderr, "error: %s: %s\n", obj.Code, obj.Message)
		return
	}
	if asJSON {
		if writeErr := encodeJSON(stdout, obj); writeErr != nil {
			_, _ = fmt.Fprintln(stderr, "error: OMF-E7006: stdout JSON write failed")
		}
		return
	}
	if obj.Cause != "" {
		_, _ = fmt.Fprintf(stderr, "error: %s: %s (cause: %s)\n", obj.Code, obj.Message, obj.Cause)
		return
	}
	_, _ = fmt.Fprintf(stderr, "error: %s: %s\n", obj.Code, obj.Message)
}
