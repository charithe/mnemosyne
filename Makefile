.PHONY: test test_all test_full clean

start_memcached:
	docker-compose up --scale memcached=3 

stop_memcached: 
	docker-compose down

test:
	go test -cover ./...

test_all: 
	go test -cover -tags=integration ./...

test_full:
	go test -race -tags=integration ./...

clean:
	go clean

