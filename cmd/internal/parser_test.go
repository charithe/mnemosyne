package internal

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCommandParsing(t *testing.T) {
	testCases := []struct {
		Name           string
		Input          string
		ExpectedOpName string
		ExpectedKey    []byte
		ExpectedValue  []byte
		ExpectedCAS    uint64
		ExpectedExpiry time.Duration
		ExpectErr      bool
	}{
		{
			Name:           "hex key values",
			Input:          "set hex:6d796b6579 hex:6d7976616c7565",
			ExpectedOpName: "set",
			ExpectedKey:    []byte("mykey"),
			ExpectedValue:  []byte("myvalue"),
		},
		{
			Name:           "base64 key values",
			Input:          "add b64:bXlrZXk b64:bXl2YWx1ZQ",
			ExpectedOpName: "add",
			ExpectedKey:    []byte("mykey"),
			ExpectedValue:  []byte("myvalue"),
		},
		{
			Name:           "string key values",
			Input:          "replace \"mykey\" \"myvalue\"",
			ExpectedOpName: "replace",
			ExpectedKey:    []byte("mykey"),
			ExpectedValue:  []byte("myvalue"),
		},
		{
			Name:           "mixed key values",
			Input:          "set hex:6d796b6579 \"another value?\"",
			ExpectedOpName: "set",
			ExpectedKey:    []byte("mykey"),
			ExpectedValue:  []byte("another value?"),
		},
		{
			Name:           "options",
			Input:          "set hex:6d796b6579 hex:6d7976616c7565 expiry:1h cas:1234",
			ExpectedOpName: "set",
			ExpectedKey:    []byte("mykey"),
			ExpectedValue:  []byte("myvalue"),
			ExpectedCAS:    1234,
			ExpectedExpiry: 1 * time.Hour,
		},
		{
			Name:      "invalid hex",
			Input:     "set hex:6d796b6579 hex:HEX",
			ExpectErr: true,
		},
		{
			Name:      "invalid base64",
			Input:     "set b64:BASE64?!! \"value\"",
			ExpectErr: true,
		},
		{
			Name:      "invalid key value",
			Input:     "set some_key \"value\"",
			ExpectErr: true,
		},
		{
			Name:      "unbalanced quotes on string",
			Input:     "set \"key\" \"value",
			ExpectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			op, err := ParseCommand(tc.Input)
			fmt.Printf("%+v\n", op)
			if tc.ExpectErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.ExpectedOpName, op.Name)
				assert.Equal(t, tc.ExpectedKey, op.Key)
				assert.Equal(t, tc.ExpectedValue, op.Value)
				assert.Equal(t, tc.ExpectedCAS, op.CAS)
				assert.Equal(t, tc.ExpectedExpiry, op.Expiry)
			}
		})
	}
}
