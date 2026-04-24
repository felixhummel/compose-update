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
```

All subdirectories are scanned recursively for Docker Compose files.


## Flags

| Flag              | Description                              | Default   |
| ----------------- | ---------------------------------------- | --------- |
| `--patch`         | Only suggest patch version updates       | `false`   |
| `--minor`         | Only suggest minor+patch version updates | `false`   |
| `-n`, `--dry-run` | Check for updates without writing        | `false`   |
| `-m`, `--max-time`| HTTP request timeout per registry call   | `5s`      |
| `-l`, `--log-level`| Log level (debug, info, warning, error) | `warning` |
| `-v`, `--version` | Show version                             | `false`   |
| `-h`, `--help`    | Show help                                | `false`   |

Without `--patch` or `--minor`, all updates (major, minor, patch) are shown.


## Installation
```bash
go install github.com/felixhummel/compose-update@latest
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
