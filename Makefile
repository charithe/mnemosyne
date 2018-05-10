.PHONY: test test_all test_full clean

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

