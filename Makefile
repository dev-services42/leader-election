

.PHONY: generate-proto
generate-proto:
	prototool generate

.PHONY: generate
generate: generate-proto
	go generate ./...

.PHONY: build
build:
	go build -o ./bin/leader-election ./cmd/leader-election

.PHONY: build-docker
build-docker:
	docker build -f Dockerfile -t devservices42/leader-election:latest .
