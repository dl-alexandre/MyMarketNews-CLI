# mpr

CLI for the USDA AMS MPR Datamart API.

## Install

Build from source:

```bash
go build -o mpr ./
```

Run:

```bash
./mpr --help
```

## Shell Completion

Generate completions and install manually:

```bash
mpr completion bash > /usr/local/etc/bash_completion.d/mpr
mpr completion zsh > /usr/local/share/zsh/site-functions/_mpr
```

## Release Checklist

1. Update version string (if present) and tag.
2. Run `go test ./...`.
3. Build release binaries for target platforms.
4. Verify `mpr completion <shell>` output.
5. Publish artifacts and checksums.
