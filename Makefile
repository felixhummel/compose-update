MAKEFLAGS += --always-make

default:
	go test ./internal/
	go build

install-dev:
	rm -f ~/.local/bin/compose-check-updates ~/.local/bin/ccu
	ln -s $$PWD/compose-check-updates  ~/.local/bin/compose-check-updates
	ln -s $$PWD/compose-check-updates  ~/.local/bin/ccu
