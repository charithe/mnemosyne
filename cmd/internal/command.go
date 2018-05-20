package internal

//go:generate ragel -G2 -o gen_parser.go parser.rl

import "time"

type Command struct {
	Name   string
	Key    []byte
	Value  []byte
	CAS    uint64
	Expiry time.Duration
}
