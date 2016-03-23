test:
	go test -v -cover $(shell go list ./... | grep -v vendor)
