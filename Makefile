.PHONY: start_memcached stop_memcached test test_all test_race bench stress clean

start_memcached:
	@docker-compose up --scale memcached=3 

stop_memcached: 
	@docker-compose down

test:
	@go test -cover ./...

test_all: 
	@go test -cover -tags=integration ./...

test_race:
	@go test -race -tags=integration ./...

bench:
	@go test -tags=integration -benchmem -bench=. ./memcache/

stress:
	@go test -c -tags=integration ./memcache 
	@stress \
		-ignore=timeout \
		./memcache.test  \
		-test.run=TestRandomLoad  \
		-req.timeout=200ms \
		-conns.per.node=5 

clean:
	@go clean

