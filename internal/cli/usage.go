package cli

const detailedUsage = `OneMoreFFmpeg - detailed usage

BASICS
  one-more-ffmpeg --json <command>        machine-readable output on stdout
  one-more-ffmpeg <command> --help        per-command help
  one-more-ffmpeg --in-detail-usage       this guide
  one-more-ffmpeg version --json          build/runtime information

DISCOVERY
  detect --json                           all discovered installations
  detect --dir C:\ffmpeg --json           one explicit directory
  validate --dir C:\ffmpeg --json         health report (always exit 0 when a report exists)
  capabilities --dir C:\ffmpeg --json     all 13 capability probes
  capabilities --dir C:\ffmpeg --only encoders --json
                                          run and output only one capability list
  --only values: muxers, demuxers, devices, decoders, encoders, filters,
                 pixel_formats, channel_layouts, sample_formats

PROVIDERS / DOWNLOAD / INSTALL
  provider list --json                    built-in official + your custom providers
  provider add --id my --url https://.../ffmpeg.zip --sha256 <64hex> --apply
      Direct mode (default): one archive URL + SHA-256 pin.
  provider add --id mydir --mode directory --name ffmpeg --url https://host/ffmpeg --apply
      Directory mode: plain static hosting; resolves latest.txt or --release-line
      and builds file names + checksum.sha256 by the documented convention.
  resolve --json > release.json           default: official FFmpeg-Builds latest
      --provider official (default) uses https://github.com/FFmpeg-Builds/FFmpeg-Builds
      latest release; defaults: gpl, static, master snapshot, x86_64 only.
      Stable lines (n9.0 ...) are not supported yet. Check the upstream README
      for OS requirements (a warning with the link is emitted).
      --provider my resolves a custom URL registered via provider add.
  download --release release.json --json
  install --release release.json --json   preview (no writes)
  install --release release.json --apply --emit-trio-lock --json
  uninstall --root <abs-install-dir> --apply --json
  uninstall --release release.json --apply --json

ENVIRONMENT / HEALTH
  env status --json                       current pointer, PATH, variables
  env set --json                          dry-run plan
  env set --apply --json                  Windows HKCU env / Linux shims + ~/.profile
  env unset --apply --json
  env rollback --journal <id> --apply --json
  --release FILE accepts UTF-8/UTF-16 JSON; PowerShell 7: Out-File -Encoding utf8
  doctor [--strict] --json                health report; exit 0 by default, or 1 with --strict when unhealthy
  journal list --json                     all transactions and pending ids
  journal rollback --journal <id> --apply --json
                                          install/uninstall/env rollback (default dry-run)

SELF-UPDATE / OUTPUT
  update --json                           check for a newer CLI (dry-run)
  update --apply --json                   download, verify SHA-256, replace this binary
  update --recover --json                 recover from an update interrupted by a crash
      --check and --apply are mutually exclusive; --recover cannot combine with either;
      amd64 only; data schema update/v1.
  --log-level debug|trace                 diagnostics on stderr (trace is more verbose)
  --quiet                                 only errors (overrides --log-level)
  Download/update progress: TTY shows animated bar (ETA), CI/pipes show plain
      stage lines, --json suppresses it; progress never writes to stdout.

SAFETY
  - Install/uninstall/rollback/provider add are dry-run by default.
  - download and capabilities cache can write without --apply.
  - install/uninstall/env writes are journaled; use journal rollback to restore.
  - uninstall moves managed directories to <data-root>/quarantine instead of deleting them.
  - Windows uses directory junction for current; pointer_file is the fallback.
  - Linux keeps shims in ~/.local/bin; --no-shell-rc avoids touching ~/.profile.
  - data root: ONE_MORE_FFMPEG_HOME, else %LOCALAPPDATA%\\OneMoreFFmpeg (Windows)
    or ~/.local/share/onemoreffmpeg (Linux).

FULL DOCUMENTATION
  docs/guides/INSTALL.md                  installation (manual binary + checksum)
  docs/guides/HOWTO.md                    beginner guide (start here)
  docs/guides/USAGE.md                    tutorial
  docs/reference/man/one-more-ffmpeg.1    man page: man ./docs/reference/man/one-more-ffmpeg.1
  docs/development/E2E_TESTING.md         Windows/Linux e2e and CI
`
