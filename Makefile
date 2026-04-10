MAKEFLAGS += --always-make

default:
	go test ./internal/
	go build

install-dev:
	rm -f ~/.local/bin/compose-update
	ln -s $$PWD/compose-update  ~/.local/bin/compose-update

e2e:
	go test -tags e2e ./tests/ -v

real-world:
	./compose-update -ldebug ~/hukudo/moni/prom
