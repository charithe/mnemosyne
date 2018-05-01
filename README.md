Mnemosyne
=========

**Work in progress**

A Go client that speaks the Memcache binary protocol.  

Usage
-----

```go
client, err := memcache.NewSimpleClient("host1:11211", "host2:11211")
// handle error
defer client.Close()

// create context
...
r, err := client.Get(ctx, []byte("key"))
if err != nil {
    fmt.Printf("Value=%v\n", r.Value())
}
```

Running Tests
-------------

Invoke `make test` to run the unit tests.


To run the integration tests, a Memcache cluster needs to be spun up using Docker Compose. 

```
make start_memcached
make test_all
make stop_memcached
```


To Do
------

- [x] Connection pooling
- [ ] Resiliency tests
- [ ] Benchmarks
- [ ] OpenCensus integration
- [ ] CLI
- [ ] Documentation
