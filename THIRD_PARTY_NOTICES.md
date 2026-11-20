# THIRD_PARTY_NOTICES

> Status: maintained; reviewed alongside v1.1.2-pre3 (2026-11-21)
> Date: 2026-11-21
> Project license: GPL-3.0-only (see root LICENSE)

This file lists Go modules recorded by `go list -m all`. License texts are copied under `third_party/licenses/` where available. v1.1.2-pre3 is the current prerelease; keep this list in sync whenever `go.mod` changes. Automated license checking via CI is not implemented yet (tracked separately, not a release blocker for v1.1.2-pre3).

| Module | Version | Detected license | License text |
|---|---|---|---|
| `github.com/dustin/go-humanize` | `v1.1.0` | MIT | `third_party/licenses/github.com_dustin_go-humanize.txt` |
| `github.com/gofrs/flock` | `v0.13.2` | BSD-3-Clause | `third_party/licenses/github.com_gofrs_flock.txt` |
| `github.com/jedib0t/go-pretty/v6` | `v6.9.2` | MIT | `third_party/licenses/github.com_jedib0t_go-pretty_v6.txt` |
| `github.com/klauspost/compress` | `v1.20.4` | BSD-3-Clause | `third_party/licenses/github.com_klauspost_compress.txt` |
| `github.com/maypok86/otter/v2` | `v2.4.0` | Apache-2.0 | `third_party/licenses/github.com_maypok86_otter_v2.txt` |
| `github.com/lmittmann/tint` | `v1.2.1` | MIT | `third_party/licenses/github.com_lmittmann_tint.txt` |
| `github.com/mattn/go-isatty` | `v0.0.24` | MIT | `third_party/licenses/github.com_mattn_go-isatty.txt` |
| `github.com/pelletier/go-toml/v2` | `v2.4.3` | MIT | `third_party/licenses/github.com_pelletier_go-toml_v2.txt` |
| `github.com/rogpeppe/go-internal` | `v1.16.0` | BSD-3-Clause | `third_party/licenses/github.com_rogpeppe_go-internal.txt` |
| `github.com/santhosh-tekuri/jsonschema/v6` | `v6.0.3` | Apache-2.0 | `third_party/licenses/github.com_santhosh-tekuri_jsonschema_v6.txt` |
| `github.com/google/go-github/v93` | `v93.0.0` | BSD-3-Clause | `third_party/licenses/github.com_google_go-github.txt` |
| `github.com/google/go-querystring` | `v1.2.0` | BSD-3-Clause | `third_party/licenses/github.com_google_go-querystring.txt` |
| `github.com/tidwall/pretty` | `v1.3.0` | MIT | `third_party/licenses/github.com_tidwall_pretty.txt` |
| `github.com/stretchr/testify` | `v1.12.1` | MIT | `third_party/licenses/github.com_stretchr_testify.txt` |
| `github.com/ulikunitz/xz` | `v0.5.17` | BSD-3-Clause | `third_party/licenses/github.com_ulikunitz_xz.txt` |
| `github.com/urfave/cli/v3` | `v3.14.0` | MIT | `third_party/licenses/github.com_urfave_cli_v3.txt` |
| `go.yaml.in/yaml/v3` | `v3.0.6` | MIT | `third_party/licenses/go.yaml.in_yaml_v3.txt` |
| `golang.org/x/sync` | `v0.25.0` | BSD-3-Clause | `third_party/licenses/golang.org_x_sync.txt` |
| `golang.org/x/sys` | `v0.50.0` | BSD-3-Clause | `third_party/licenses/golang.org_x_sys.txt` |
| `golang.org/x/term` | `v0.48.0` | BSD-3-Clause | `third_party/licenses/golang.org_x_term.txt` |
| `github.com/clipperhouse/uax29/v2` | `v2.7.0` | BSD-3-Clause | `third_party/licenses/github.com_clipperhouse_uax29_v2.txt` |
| `github.com/dlclark/regexp2` | `v1.12.0` | MIT | `third_party/licenses/github.com_dlclark_regexp2.txt` |
| `github.com/mattn/go-runewidth` | `v0.0.31` | MIT | `third_party/licenses/github.com_mattn_go-runewidth.txt` |
| `github.com/rivo/uniseg` | `v0.4.7` | MIT | `third_party/licenses/github.com_rivo_uniseg.txt` |
| `golang.org/x/text` | `v0.44.0` | BSD-3-Clause | `third_party/licenses/golang.org_x_text.txt` |
| `golang.org/x/tools` | `v0.52.0` | BSD-3-Clause | `third_party/licenses/golang.org_x_tools.txt` |

Detection is based on the first 20 lines of each upstream license file. It is an engineering aid, not legal advice.

The project itself is licensed under GPL-3.0-only; see `LICENSE` and `docs/development/adr/0003-license.md`.
