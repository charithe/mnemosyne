Mnemosyne
=========

[![GoDoc](https://godoc.org/github.com/charithe/mnemosyne/memcache?status.svg)](https://godoc.org/github.com/charithe/mnemosyne/memcache)

**Work in progress**

A Go client that speaks the Memcache binary protocol.  

Usage
-----

```go
client, err := memcache.NewSimpleClient("host1:11211", "host2:11211")
...
// handle error
...
defer client.Close()
...
// create context
...
if r, err := client.Get(ctx, []byte("key")); err == nil {
    fmt.Printf("Value=%v\n", r.Value())
}
```

### Interpreting return values

Single command operations such as `GET` produce a `memcache.Result` and an `error` value. Multiple command operations
such as `MGET` produce an array of `memcache.Result` objects and a single `error` value.

If the `error` is nil, the operation has succeeded. Accessing the `Value()` or `NumericValue()` methods of the resulting
`Result` objects should return the response returned from the server.

If the `error` is not nil and is an instance of `*memcache.Error`, some or all of the commands in the batch may have failed.
Ensure that the `Err()` method returns nil for each result before trusting the output of `Value()` or `NumericValue()`.



Development
-----------

This project uses [dep](https://golang.github.io/dep/) for dependency management.

After checking out the source, run `dep ensure` to download the dependencies.


### Running Tests

Invoke `make test` to run the unit tests.


To run the integration tests, a Memcache cluster needs to be spun up using [Docker Compose](https://docs.docker.com/compose/) 

```
make start_memcached
make test_all
make stop_memcached
```


To Do
------

- [x] Connection pooling
- [x] Resiliency tests
- [x] OpenCensus metrics
- [ ] OpenCensus traces
- [ ] CLI
- [ ] Stress tests
