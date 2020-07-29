.PHONY: test
test:
	clear
	go test -count=1 -timeout 10s -coverprofile=./cover/all-profile.out -covermode=set -coverpkg=./... ./...; \
	go tool cover -html=./cover/all-profile.out -o ./cover/all-coverage.html

lint:
	golangci-lint run ./...
