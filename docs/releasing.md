# Install script and releases

There is no release yet, so the install script has nothing to download. Until there is one,
install from a checkout with `go install ./cmd/ym`.

## Install a release

Release binaries support Apple Silicon and Intel Macs. They need no Go and no source checkout.
The installer checks the checksums:

```sh
curl -fsSL https://raw.githubusercontent.com/mateuszjasiuk/yardmaster/main/scripts/install.sh | sh
```

It writes to `~/.local/bin` by default. Set `YM_INSTALL_DIR` (or pass `--bin-dir DIR`) to use a
different folder, or download the script first if you don't want to pipe it into a shell.
`--version v0.1.0` installs a given release. Check the installed build with `ym version`.

Release files can be verified with the GitHub CLI:

```sh
gh attestation verify ym_Darwin_arm64.tar.gz --repo mateuszjasiuk/yardmaster
```

## Make a release

GitHub Actions builds releases from semantic version tags. They start as drafts, so the archives
and attestations can be checked before publishing.

1. Make sure the release commit is on `main`, then push a tag such as `v0.1.0`.
2. Check the draft release: its two macOS archives, checksums, generated notes and attestations.
   Then publish it.
3. After the first release that works end to end, turn on immutable releases in the repository
   settings.
