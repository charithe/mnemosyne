package memcache

import (
	"context"
	"log"
	"time"
)

func Example() {
	client, err := NewSimpleClient("localhost:11211")
	if err != nil {
		log.Fatalf("Failed to start client for resiliency test: %+v", err)
	}
	defer client.Close()

	ctx, cancelFunc := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelFunc()

	// simple SET
	result, err := client.Set(ctx, []byte("key1"), []byte("value1"))
	if err != nil {
		log.Fatalf("Failed to SET key: %+v", err)
	}

	// use the CAS ID to REPLACE the value
	_, err = client.Replace(ctx, []byte("key1"), []byte("value1_1"), WithCASValue(result.CAS()))
	if err != nil {
		log.Fatalf("Failed to REPLACE key: %+v", err)
	}

	// GET the value
	result, err = client.Get(ctx, []byte("key1"))
	if err != nil {
		if result.Err() == ErrKeyNotFound {
			log.Printf("Key not found")
		} else {
			log.Fatalf("Failed to GET key: %+v", err)
		}
	}

	log.Printf("Got value: %s", result.Value())

	// SET key with expiry
	result, err = client.Set(ctx, []byte("key2"), []byte("value2"), WithExpiry(1*time.Hour))
	if err != nil {
		log.Fatalf("Failed to SET key: %+v", err)
	}

	// MGET
	results, err := client.MultiGet(ctx, []byte("key1"), []byte("key2"))
	if err != nil {
		log.Printf("Failed to MGET: %+v", err)
	}

	for _, r := range results {
		log.Printf("Key=%s Value=%s", r.Key(), r.Value())
	}
}

func ExampleNewClient() {
	// Connect to a cluster with 3 parallel connections per node
	client, err := NewClient(WithNodePicker(NewSimpleNodePicker("localhost:11211", "localhost:11212", "localhost:11213")), WithConnectionsPerNode(3))
	if err != nil {
		log.Fatalf("Failed to create client: %+v", err)
	}
	defer client.Close()

	client.Set(context.Background(), []byte("key"), []byte("value"))
}
