.PHONY: test test_all test_full clean

test:
	go test ./...

test_all: 
	go test -tags=integration ./...

test_full:
	go test -race -tags=integration ./...

clean:
	go clean

