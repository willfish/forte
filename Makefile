.PHONY: dev build test lint clean

dev:
	wails3 dev -config ./build/config.yml

build:
	task build

test:
	go test ./...

lint:
	golangci-lint run
	cd frontend && npm run check

clean:
	rm -rf bin/ frontend/dist/ .task/
