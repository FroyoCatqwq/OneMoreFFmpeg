# THIRD_PARTY_NOTICES

> Status: Phase 1 initial draft
> Date: 2026-11-10
> Project license: GPL-3.0-only (see root LICENSE)

This file lists Go modules recorded by `go list -m all` for Phase 1. License texts are copied under `third_party/licenses/` where available. Final license review happens before the first public binary release.

| Module | Version | Detected license | License text |
|---|---|---|---|
| `github.com/dustin/go-humanize` | `v1.1.0` | MIT | `third_party/licenses/github.com_dustin_go-humanize.txt` |
| `github.com/klauspost/compress` | `v1.20.4` | BSD-3-Clause | `third_party/licenses/github.com_klauspost_compress.txt` |
| `github.com/lmittmann/tint` | `v1.2.1` | MIT | `third_party/licenses/github.com_lmittmann_tint.txt` |
| `github.com/mattn/go-isatty` | `v0.0.24` | MIT | `third_party/licenses/github.com_mattn_go-isatty.txt` |
| `github.com/pelletier/go-toml/v2` | `v2.4.3` | MIT | `third_party/licenses/github.com_pelletier_go-toml_v2.txt` |
| `github.com/sethvargo/go-retry` | `v0.5.0` | Apache-2.0 | `third_party/licenses/github.com_sethvargo_go-retry.txt` |
| `github.com/stretchr/testify` | `v1.12.1` | MIT | `third_party/licenses/github.com_stretchr_testify.txt` |
| `github.com/ulikunitz/xz` | `v0.5.17` | BSD-3-Clause | `third_party/licenses/github.com_ulikunitz_xz.txt` |
| `github.com/urfave/cli/v3` | `v3.14.0` | MIT | `third_party/licenses/github.com_urfave_cli_v3.txt` |
| `go.yaml.in/yaml/v3` | `v3.0.5` | MIT | `third_party/licenses/go.yaml.in_yaml_v3.txt` |
| `golang.org/x/sync` | `v0.25.0` | BSD-3-Clause | `third_party/licenses/golang.org_x_sync.txt` |
| `golang.org/x/sys` | `v0.50.0` | BSD-3-Clause | `third_party/licenses/golang.org_x_sys.txt` |
| `golang.org/x/term` | `v0.48.0` | BSD-3-Clause | `third_party/licenses/golang.org_x_term.txt` |
| `golang.org/x/time` | `v0.16.0` | BSD-3-Clause | `third_party/licenses/golang.org_x_time.txt` |

Detection is based on the first 20 lines of each upstream license file. It is an engineering aid, not legal advice.

The project itself is licensed under GPL-3.0-only; see `LICENSE` and `docs/LICENSE_DECISION.md`.
