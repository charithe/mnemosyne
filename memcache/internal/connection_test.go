package internal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadPacket(t *testing.T) {
	cw, mock := newMockConnWrapper()

	packets := []byte{
		0x81, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x09,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x4e, 0x6f, 0x74, 0x20,
		0x66, 0x6f, 0x75, 0x6e,
		0x64,
		0x81, 0x00, 0x00, 0x00,
		0x04, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x09,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0xde, 0xad, 0xbe, 0xef,
		0x57, 0x6f, 0x72, 0x6c,
		0x64,
		0x81, 0x00, 0x00, 0x05,
		0x04, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x0E,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0xde, 0xad, 0xbe, 0xef,
		0x48, 0x65, 0x6c, 0x6c,
		0x6f, 0x57, 0x6f, 0x72,
		0x6c, 0x64,
		0x81, 0x02, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
	}

	assert.NoError(t, mock.setPayload(packets))

	mkContext := func() (context.Context, time.Time) {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		deadline, _ := ctx.Deadline()
		return ctx, deadline
	}

	t.Run("error response", func(t *testing.T) {
		// error response
		ctx, deadline := mkContext()
		pkt, err := cw.ReadPacket(ctx)
		assert.NoError(t, err)
		assert.Equal(t, deadline, mock.readDeadline)
		assert.Equal(t, StatusKeyNotFound, pkt.StatusCode)
		assert.Equal(t, uint64(0), pkt.CAS)
		assert.Nil(t, pkt.Extras)
		assert.Nil(t, pkt.Key)
		assert.Equal(t, uint32(0), pkt.Opaque)
		assert.Equal(t, "Not found", string(pkt.Value))
	})

	t.Run("get response", func(t *testing.T) {
		// GET response
		ctx, deadline := mkContext()
		pkt, err := cw.ReadPacket(ctx)
		assert.NoError(t, err)
		assert.Equal(t, deadline, mock.readDeadline)
		assert.Equal(t, StatusNoError, pkt.StatusCode)
		assert.Equal(t, uint64(1), pkt.CAS)
		assert.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, pkt.Extras)
		assert.Nil(t, pkt.Key)
		assert.Equal(t, uint32(0), pkt.Opaque)
		assert.Equal(t, "World", string(pkt.Value))
	})

	t.Run("getk response", func(t *testing.T) {
		// GETK response
		ctx, deadline := mkContext()
		pkt, err := cw.ReadPacket(ctx)
		assert.NoError(t, err)
		assert.Equal(t, deadline, mock.readDeadline)
		assert.Equal(t, StatusNoError, pkt.StatusCode)
		assert.Equal(t, uint64(1), pkt.CAS)
		assert.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, pkt.Extras)
		assert.Equal(t, "Hello", string(pkt.Key))
		assert.Equal(t, uint32(0), pkt.Opaque)
		assert.Equal(t, "World", string(pkt.Value))
	})

	t.Run("set response", func(t *testing.T) {
		// SET response
		ctx, deadline := mkContext()
		pkt, err := cw.ReadPacket(ctx)
		assert.NoError(t, err)
		assert.Equal(t, deadline, mock.readDeadline)
		assert.Equal(t, StatusNoError, pkt.StatusCode)
		assert.Equal(t, uint64(1), pkt.CAS)
		assert.Nil(t, pkt.Extras)
		assert.Nil(t, pkt.Key)
		assert.Equal(t, uint32(0), pkt.Opaque)
		assert.Nil(t, pkt.Value)
	})
}

func TestWritePacket(t *testing.T) {
	cw, mock := newMockConnWrapper()

	mkContext := func() (context.Context, time.Time) {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		deadline, _ := ctx.Deadline()
		return ctx, deadline
	}

	testCases := []struct {
		name          string
		pkt           *Request
		expectedBytes []byte
	}{
		{
			name: "get request", // key only
			pkt: &Request{
				OpCode: OpGet,
				Key:    []byte("Hello"),
			},
			expectedBytes: []byte{
				0x80, 0x00, 0x00, 0x05,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x05,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x48, 0x65, 0x6c, 0x6c,
				0x6f,
			},
		},
		{
			name: "add request", // key, value, and extras
			pkt: &Request{
				OpCode: OpAdd,
				Key:    []byte("Hello"),
				Value:  []byte("World"),
				Extras: []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x00, 0x1c, 0x20},
			},
			expectedBytes: []byte{
				0x80, 0x02, 0x00, 0x05,
				0x08, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x12,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0xde, 0xad, 0xbe, 0xef,
				0x00, 0x00, 0x1c, 0x20,
				0x48, 0x65, 0x6c, 0x6c,
				0x6f, 0x57, 0x6f, 0x72,
				0x6c, 0x64,
			},
		},
		{
			name: "stat request", // No key, value, or extras
			pkt: &Request{
				OpCode: OpStat,
			},
			expectedBytes: []byte{
				0x80, 0x10, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			mock.Reset()
			ctx, deadline := mkContext()
			assert.NoError(t, cw.WritePacket(ctx, tc.pkt))
			assert.NoError(t, cw.FlushWriteBuffer())
			assert.Equal(t, deadline, mock.writeDeadline)

			actualBytes := mock.getPayload()
			assert.Equal(t, tc.expectedBytes, actualBytes)
		})
	}
}
