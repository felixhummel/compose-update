# compose-update
Checks Docker Compose images for newer versions.


## Usage
```bash
# Check current directory (all versions: major, minor, patch)
compose-update
# Check a specific directory
compose-update /path/to/project
# Patch updates only
compose-update --patch
# Minor updates only (includes patch)
compose-update --minor
# Dry-run: check without writing changes
compose-update --dry-run
# include images
compose-update --glob caddy
# skip images
compose-update --exclude "postgres:*"
# machine-readable output (one JSON object per line)
compose-update --dry-run --output jsonl
```

`--output jsonl` prints one object per update:
```json
{"path":"docker-compose.yml","old":"postgres:14.5","new":"postgres:18.4"}
```
`path` is empty when checking a single image via `--image`. Unlike `text`, `jsonl` also
reports updates that were written to disk (i.e. without `--dry-run`).

All subdirectories are scanned recursively for Docker Compose files.


## Flags
```
$ compose-update --help
Usage: compose-update [flags] [directory]

Arguments:
  directory     Root directory to scan for Docker Compose files (default: ".")

Flags:
  -n, --dry-run               Only check for updates, do not write
  -x, --exclude stringArray   Image globs to exclude (repeatable)
  -g, --glob stringArray      Image globs to include as whitelist (repeatable)
  -h, --help                  Show help message
      --image string          Check a single image (e.g. nginx:1.25.0)
  -l, --log-level string      Log level (debug, info, warning, error) (default "warning")
      --major                 Include major version updates
  -m, --max-time duration     HTTP request timeout per registry call (default 5s)
      --minor                 Only update to the latest minor version
  -o, --output string         Output format (text, jsonl) (default "text")
      --patch                 Only update to the latest patch version
      --tags string           Print all tags for an image (e.g. postgres:14.5)
  -v, --version               Show version information
```

## Installation
```bash
go install github.com/felixhummel/compose-update@latest
# or
mise use -g github:felixhummel/compose-update
```

Or build from source:

```bash
git clone https://github.com/felixhummel/compose-update.git
cd compose-update
make
make install-dev  # symlinks binary to ~/.local/bin/compose-update
```


## How it works
`compose-update` scans for Docker Compose files, reads each service's image tag, and queries the container registry for newer semver versions. Image tags are updated in-place.

**Docker Hub** (`docker.io`): resolves the `latest` tag's manifest digest, then finds the semver tags sharing that digest — the same approach Docker itself uses to publish a versioned release alongside `latest`. Falls back to filtering by `v`-prefix or full tag enumeration if no digest match is found.

**GitHub Container Registry** (`ghcr.io`): calls the GitHub Releases API (`/releases/latest`) to get the current release tag directly, avoiding tag-list pagination entirely.

**Other registries**: paginates the OCI `tags/list` endpoint, stopping early once pages no longer contain semver tags.
