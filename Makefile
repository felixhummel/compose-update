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

pre-release: default

jj-update-main:
	jj bookmark set main -r @-

push-with-tags: jj-update-main
	git push
	git push --tags

major-release: pre-release
	bump-my-version bump major
	make push-with-tags

minor-release: pre-release
	bump-my-version bump minor
	make push-with-tags

patch-release: pre-release
	bump-my-version bump patch
	make push-with-tags
