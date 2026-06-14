.PHONY: dev build test lint check clean

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

clean:
	rm -rf bin/ frontend/dist/ .task/
