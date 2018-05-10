Mnemosyne
=========

[![GoDoc](https://godoc.org/github.com/charithe/mnemosyne/memcache?status.svg)](https://godoc.org/github.com/charithe/mnemosyne/memcache)

**Work in progress**

A Memcache binary protocol client designed to improve observability and bounded execution times. 

Usage
-----

See [godocs](https://godoc.org/github.com/charithe/mnemosyne/memcache) for a full example and information about client parameters.


```go
client, err := memcache.NewSimpleClient("host1:11211", "host2:11211")
...
...
r, err := client.Get(ctx, []byte("key"))
if err != nil {
    if r.Err() == memcache.ErrKeyNotFound {
        log.Printf("Cache miss")
    }  
} else {
    log.Printf("Cache hit [value=%x]", r.Value())
}
```

### Interpreting return values

Single command operations such as `GET` produce a `memcache.Result` and an `error` value. Multiple command operations
such as `MGET` produce an array of `memcache.Result` objects and a single `error` value.

If the `error` is nil, the operation has succeeded. Accessing the `Value()` or `NumericValue()` methods of the resulting
`Result` objects should return the response returned from the server.

If the `error` is not nil and is an instance of `*memcache.Error`, some or all of the commands in the batch may have failed.
Ensure that the `Err()` method returns nil for each result before trusting the output of `Value()` or `NumericValue()`.


### OpenCensus Metrics

The `ocmemcache` package contains view definitions for all of the metrics collected by the library. Easiest way to 
get all of the metrics at once is to register the `ocmemcache.DefaultViews` array.

```go
view.Register(ocmemcache.DefaultViews...)
```


Development
-----------

This project uses [dep](https://golang.github.io/dep/) for dependency management. After checking out the source, 
run `dep ensure` to download the dependencies.


### Running Tests

Invoke `make test` to run the unit tests. 


To run the integration tests, a Memcache cluster needs to be spun up using [Docker Compose](https://docs.docker.com/compose/) 

```
make start_memcached
make test_all
make stop_memcached
```


For stress testing, the `stress` utility must be installed:

```
go get -u golang.org/x/tools/cmd/stress
make stress
```

To Do
------

- [x] Connection pooling
- [x] Resiliency tests
- [x] OpenCensus metrics
- [ ] OpenCensus traces
- [ ] CLI
- [x] Stress tests
