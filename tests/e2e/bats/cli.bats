#!/usr/bin/env bats

setup() {
  : "${OMF_BINARY:?set OMF_BINARY to the built CLI}"
  export ONE_MORE_FFMPEG_HOME="$BATS_TEST_TMPDIR/data-root"
}

@test "version --json returns v1 envelope" {
  run "$OMF_BINARY" version --json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"schema":"onemoreffmpeg.result/v1"'* ]]
}

@test "provider list works offline" {
  run "$OMF_BINARY" provider list --json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"btbn"'* ]]
}

@test "doctor always returns a report" {
  run "$OMF_BINARY" doctor --json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"onemoreffmpeg.doctor/v1"'* ]]
}

@test "doctor --strict exits nonzero when unhealthy" {
  run "$OMF_BINARY" doctor --strict --json
  [ "$status" -eq 1 ]
  [[ "$output" == *'OMF-E6006'* ]]
}

@test "env system scope with consent is a dry-run by default" {
  run "$OMF_BINARY" env set --system --authorize-system --json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"dry_run":true'* ]]
}
