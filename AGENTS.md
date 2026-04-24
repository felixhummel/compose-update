# compose-update
This is a go project that checks docker-compose.yml images for newer versions.
Run `go run main.go --help` to see the UI.
To see it in action:
```
go run main.go --dry-run --log-level=info tests/docker-compose.yml
```
