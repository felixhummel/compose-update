MAKEFLAGS += --always-make

default:
	go test ./internal/
	go build

install-dev:
	rm -f ~/.local/bin/compose-update
	ln -s $$PWD/compose-update  ~/.local/bin/compose-update

real-world:
	./compose-update -ldebug ~/hukudo/moni/prom
