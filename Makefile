build:
	@go build -o bin/app

run: build
	@./bin/app

test:
	@go test -v ./...

clean:
	@./scripts/clean_network_folders.sh
