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

RELEASE CATALOG
  release list --channel stable --json
      Stable versions come from https://ffmpeg.org/download.html#releases.
      A line such as 9.0 is displayed as its newest patch (for example 9.0.2);
      the downloadable asset is resolved from the provider release URL.
  release list --channel daily --days 7 --count 3 --json
      Daily/snapshot builds default to the last 7 days; --days changes the
      lookback and --count limits how many snapshots are displayed.
  release install --channel stable --version 9.0.2 --apply
  release install --channel daily --snapshot autobuild-YYYY-MM-DD-HH-MM --apply
      The newest displayed version is used when --version/--snapshot is omitted.
      All installs are dry-run unless --apply is given.
  release install --provider bnqdzj-nonfree --allow-nonfree --apply
      nonfree providers require explicit consent.

PROVIDERS / DOWNLOAD / INSTALL
  provider list --json                    built-in and custom providers
  provider add --id my --url https://.../ffmpeg.zip --sha256 <64hex> --apply
  resolve --provider btbn --channel stable --target win64 --json > release.json
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
  doctor --json                           health report; always exit 0, read healthy
  journal list --json                     all transactions and pending ids
  journal rollback --journal <id> --apply --json
                                          install/uninstall/env rollback (default dry-run)

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
  docs/HOWTO.md                           beginner guide (start here)
  docs/USAGE.md                           tutorial
  docs/man/one-more-ffmpeg.1              man page: man ./docs/man/one-more-ffmpeg.1
  docs/E2E_TESTING.md                     Windows/Linux e2e and CI
`
