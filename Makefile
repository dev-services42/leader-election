

.PHONY: generate-proto
generate-proto:
	prototool generate

.PHONY: generate
generate: generate-proto
	go generate ./...
