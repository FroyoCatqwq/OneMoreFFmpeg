// Package deps pins planned future-phase dependencies in go.mod.
//
// The blank imports live in future.go behind the "deps" build tag. Normal
// builds do not link those packages, but "go mod tidy" still records them.
package deps
