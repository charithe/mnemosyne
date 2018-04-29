package memcache

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/charithe/mnemosyne/memcache/internal"
)

var (
	ErrInvalidClientConfig = errors.New("Invalid client configuration")
	ErrNoResult            = errors.New("No result")
	ErrClientShutdown      = errors.New("Client shutdown")

	ErrKeyNotFound                   = errors.New("Key not found")
	ErrKeyExists                     = errors.New("Key exists")
	ErrValueTooLarge                 = errors.New("Value too large")
	ErrInvalidArguments              = errors.New("Invalid arguments")
	ErrItemNotStored                 = errors.New("Item not stored")
	ErrIncrOnNonNumericValue         = errors.New("Incr on non-numeric value")
	ErrVBucketBelongsToAnotherServer = errors.New("VBucket belongs to another server")
	ErrAuthenticationError           = errors.New("Authentication error")
	ErrUnknownCommand                = errors.New("Unknown command")
	ErrOutOfMemory                   = errors.New("Out of memory")
	ErrNotSupported                  = errors.New("Not supported")
	ErrInternalError                 = errors.New("Internal error")
	ErrBusy                          = errors.New("Busy")
	ErrTemporaryFailure              = errors.New("Temporary failure")
)

var statusCodeToErr = map[uint16]error{
	internal.StatusNoError:                       nil,
	internal.StatusKeyNotFound:                   ErrKeyNotFound,
	internal.StatusKeyExists:                     ErrKeyExists,
	internal.StatusValueTooLarge:                 ErrValueTooLarge,
	internal.StatusInvalidArguments:              ErrInvalidArguments,
	internal.StatusItemNotStored:                 ErrItemNotStored,
	internal.StatusIncrOnNonNumericValue:         ErrIncrOnNonNumericValue,
	internal.StatusVBucketBelongsToAnotherServer: ErrVBucketBelongsToAnotherServer,
	internal.StatusAuthenticationError:           ErrAuthenticationError,
	internal.StatusUnknownCommand:                ErrUnknownCommand,
	internal.StatusOutOfMemory:                   ErrOutOfMemory,
	internal.StatusNotSupported:                  ErrNotSupported,
	internal.StatusInternalError:                 ErrInternalError,
	internal.StatusBusy:                          ErrBusy,
	internal.StatusTemporaryFailure:              ErrTemporaryFailure,
}

type expiryFunc func(expiry time.Duration) []byte

var (
	// normal expiry function
	stdExpiryFunc = func(expiry time.Duration) []byte {
		extras := internal.NewBuf()
		extras.WriteUint32(uint32(expiry.Seconds()))
		return extras.Bytes()
	}

	// expiry function that does nothing
	nopExpiryFunc = func(expiry time.Duration) []byte {
		return nil
	}

	// expiry function paired with zero flag value
	flagExpiryFunc = func(expiry time.Duration) []byte {
		extras := internal.NewBuf()
		extras.WriteUint32(0)
		extras.WriteUint32(uint32(expiry.Seconds()))
		return extras.Bytes()
	}

	opCodeToExpiry = map[uint8]expiryFunc{
		internal.OpAppend:   nopExpiryFunc,
		internal.OpPrepend:  nopExpiryFunc,
		internal.OpGet:      flagExpiryFunc,
		internal.OpGetQ:     flagExpiryFunc,
		internal.OpGetK:     flagExpiryFunc,
		internal.OpGetKQ:    flagExpiryFunc,
		internal.OpSet:      flagExpiryFunc,
		internal.OpSetQ:     flagExpiryFunc,
		internal.OpAdd:      flagExpiryFunc,
		internal.OpAddQ:     flagExpiryFunc,
		internal.OpReplace:  flagExpiryFunc,
		internal.OpReplaceQ: flagExpiryFunc,
	}
)

// Set the extras field based on the op code
func setExtras(req *internal.Request, expiry time.Duration) {
	f, ok := opCodeToExpiry[req.OpCode]
	if !ok {
		req.Extras = stdExpiryFunc(expiry)
		return
	}

	req.Extras = f(expiry)
}

// result implements the Result interface
type result struct {
	resp *internal.Response
}

func (r *result) Key() []byte {
	return r.resp.Key
}

func (r *result) Value() []byte {
	return r.resp.Value
}

func (r *result) NumericValue() uint64 {
	return binary.BigEndian.Uint64(r.resp.Value)
}

func (r *result) CAS() uint64 {
	return r.resp.CAS
}

func (r *result) Err() error {
	if r.resp.Err != nil {
		return r.resp.Err
	}

	return statusCodeToErr[r.resp.StatusCode]
}

func (r *result) String() string {
	return fmt.Sprintf("%+v", r.resp)
}

// Helper functions for creating Result objects
func newResult(resp *internal.Response) Result {
	return &result{resp: resp}
}

func newErrResult(err error) Result {
	return &result{resp: &internal.Response{Err: err}}
}

// MutationOpt is an optional modification to apply to the request
type MutationOpt func(req *internal.Request)

// WithExpiry sets an expiry time for the key being mutated
func WithExpiry(expiry time.Duration) MutationOpt {
	return func(req *internal.Request) {
		setExtras(req, expiry)
	}
}

// WithCASValue sets the CAS value to be used in the request
func WithCASValue(cas uint64) MutationOpt {
	return func(req *internal.Request) {
		req.CAS = cas
	}
}

// keyError represents an error with an individual key
type KeyError struct {
	Key []byte
	Err error
}

func (ke *KeyError) Error() string {
	return fmt.Sprintf("Key=%x Error=%v", ke.Key, ke.Err)
}

// create an Error from a list of errors
func newErrorFromErrList(errors ...error) error {
	return &Error{errors: errors}
}

// create an Error from the message
func newErrorWithMessage(msg string) error {
	return &Error{msg: msg}
}

// Error is a generic error type providing information about individual failures
type Error struct {
	errors []error
	msg    string
}

func (e *Error) Error() string {
	if e.msg == "" {
		b := new(bytes.Buffer)
		for i, err := range e.errors {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(err.Error())
		}
		e.msg = b.String()
	}

	return e.msg
}

func (e *Error) Errors() []error {
	return e.errors
}
