.PHONY: dev build test lint check changelog release-notes clean

dev:
	wails3 dev -config ./build/config.yml

build:
	task build

test:
	go test -tags nocgo ./...

lint:
	golangci-lint run --build-tags nocgo
	cd frontend && npm run check

check: test lint

changelog:
	git cliff --output CHANGELOG.md

release-notes:
	git cliff --latest --strip header --output RELEASE_NOTES.md

clean:
	rm -rf bin/ frontend/dist/ .task/
