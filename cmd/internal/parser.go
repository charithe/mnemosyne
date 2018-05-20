
//line cmd/internal/parser.rl:1
package internal

import (
    "fmt"
    "encoding/base64"
    "encoding/hex"
    "time"
    "strconv"
)


//line cmd/internal/parser.go:15
const mnemosyne_start int = 1
const mnemosyne_first_final int = 48
const mnemosyne_error int = 0

const mnemosyne_en_inside_string int = 53
const mnemosyne_en_main int = 1


//line cmd/internal/parser.rl:14


func ParseCommand(cmdLine string) (cmd *Command, err error) {
    defer func(){
        if r := recover(); r != nil {
            if e, ok := r.(error); ok {
                err = e
            }
        }
    }()

    var p, pe, cs, te, ts, act, top int
    var stack [8]int

    data := []byte(cmdLine)
    eof := len(data)
    pe = eof

    var mark int
    var valueHolder []byte

    decodeBase64 := func(encodedVal []byte) []byte {
        decodedLen := base64.RawStdEncoding.DecodedLen(len(encodedVal))
        decodedVal := make([]byte, decodedLen)
        if _, err := base64.RawStdEncoding.Decode(decodedVal, encodedVal); err != nil {
            panic(err)
        }

        return decodedVal
    }

    decodeHex := func(encodedVal []byte) []byte {
        decodedVal, err := hex.DecodeString(string(encodedVal))
        if err != nil {
            panic(err)
        }

        return decodedVal
    }

    parseExpiry := func(expiryVal []byte) time.Duration {
        expiry, err := time.ParseDuration(string(expiryVal))
        if err != nil {
            panic(err)
        }

        return expiry
    }

    parseCAS := func(casVal []byte) uint64 {
        cas, err := strconv.ParseUint(string(casVal), 10, 64)
        if err != nil {
            panic(err)
        }

        return cas
    }

    
//line cmd/internal/parser.go:84
	{
	cs = mnemosyne_start
	top = 0
	ts = 0
	te = 0
	act = 0
	}

//line cmd/internal/parser.go:93
	{
	if p == pe {
		goto _test_eof
	}
	goto _resume

_again:
	switch cs {
	case 1:
		goto st1
	case 0:
		goto st0
	case 2:
		goto st2
	case 3:
		goto st3
	case 4:
		goto st4
	case 5:
		goto st5
	case 6:
		goto st6
	case 7:
		goto st7
	case 8:
		goto st8
	case 48:
		goto st48
	case 9:
		goto st9
	case 10:
		goto st10
	case 11:
		goto st11
	case 12:
		goto st12
	case 13:
		goto st13
	case 49:
		goto st49
	case 14:
		goto st14
	case 15:
		goto st15
	case 16:
		goto st16
	case 17:
		goto st17
	case 18:
		goto st18
	case 19:
		goto st19
	case 20:
		goto st20
	case 21:
		goto st21
	case 50:
		goto st50
	case 22:
		goto st22
	case 23:
		goto st23
	case 24:
		goto st24
	case 25:
		goto st25
	case 51:
		goto st51
	case 26:
		goto st26
	case 27:
		goto st27
	case 28:
		goto st28
	case 29:
		goto st29
	case 52:
		goto st52
	case 30:
		goto st30
	case 31:
		goto st31
	case 32:
		goto st32
	case 33:
		goto st33
	case 34:
		goto st34
	case 35:
		goto st35
	case 36:
		goto st36
	case 37:
		goto st37
	case 38:
		goto st38
	case 39:
		goto st39
	case 40:
		goto st40
	case 41:
		goto st41
	case 42:
		goto st42
	case 43:
		goto st43
	case 44:
		goto st44
	case 45:
		goto st45
	case 46:
		goto st46
	case 47:
		goto st47
	case 53:
		goto st53
	case 54:
		goto st54
	}

	if p++; p == pe {
		goto _test_eof
	}
_resume:
	switch cs {
	case 1:
		goto st_case_1
	case 0:
		goto st_case_0
	case 2:
		goto st_case_2
	case 3:
		goto st_case_3
	case 4:
		goto st_case_4
	case 5:
		goto st_case_5
	case 6:
		goto st_case_6
	case 7:
		goto st_case_7
	case 8:
		goto st_case_8
	case 48:
		goto st_case_48
	case 9:
		goto st_case_9
	case 10:
		goto st_case_10
	case 11:
		goto st_case_11
	case 12:
		goto st_case_12
	case 13:
		goto st_case_13
	case 49:
		goto st_case_49
	case 14:
		goto st_case_14
	case 15:
		goto st_case_15
	case 16:
		goto st_case_16
	case 17:
		goto st_case_17
	case 18:
		goto st_case_18
	case 19:
		goto st_case_19
	case 20:
		goto st_case_20
	case 21:
		goto st_case_21
	case 50:
		goto st_case_50
	case 22:
		goto st_case_22
	case 23:
		goto st_case_23
	case 24:
		goto st_case_24
	case 25:
		goto st_case_25
	case 51:
		goto st_case_51
	case 26:
		goto st_case_26
	case 27:
		goto st_case_27
	case 28:
		goto st_case_28
	case 29:
		goto st_case_29
	case 52:
		goto st_case_52
	case 30:
		goto st_case_30
	case 31:
		goto st_case_31
	case 32:
		goto st_case_32
	case 33:
		goto st_case_33
	case 34:
		goto st_case_34
	case 35:
		goto st_case_35
	case 36:
		goto st_case_36
	case 37:
		goto st_case_37
	case 38:
		goto st_case_38
	case 39:
		goto st_case_39
	case 40:
		goto st_case_40
	case 41:
		goto st_case_41
	case 42:
		goto st_case_42
	case 43:
		goto st_case_43
	case 44:
		goto st_case_44
	case 45:
		goto st_case_45
	case 46:
		goto st_case_46
	case 47:
		goto st_case_47
	case 53:
		goto st_case_53
	case 54:
		goto st_case_54
	}
	goto st_out
	st1:
//line NONE:1
ts = 0

		if p++; p == pe {
			goto _test_eof1
		}
	st_case_1:
//line cmd/internal/parser.go:339
		switch data[p] {
		case 32:
			goto tr0
		case 97:
			goto tr2
		case 114:
			goto tr3
		case 115:
			goto tr4
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr0
		}
		goto st0
st_case_0:
	st0:
		cs = 0
		goto _out
tr0:
//line cmd/internal/parser.rl:75

            cmd = &Command{}
        
	goto st2
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
//line cmd/internal/parser.go:369
		switch data[p] {
		case 32:
			goto st2
		case 97:
			goto tr6
		case 114:
			goto tr7
		case 115:
			goto tr8
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto st2
		}
		goto st0
tr2:
//line cmd/internal/parser.rl:75

            cmd = &Command{}
        
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st3
tr6:
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st3
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
//line cmd/internal/parser.go:405
		if data[p] == 100 {
			goto st4
		}
		goto st0
	st4:
		if p++; p == pe {
			goto _test_eof4
		}
	st_case_4:
		if data[p] == 100 {
			goto st5
		}
		goto st0
	st5:
		if p++; p == pe {
			goto _test_eof5
		}
	st_case_5:
		if data[p] == 32 {
			goto tr11
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr11
		}
		goto st0
tr11:
//line cmd/internal/parser.rl:83

            cmd.Name = string(data[mark:p])
        
	goto st6
	st6:
		if p++; p == pe {
			goto _test_eof6
		}
	st_case_6:
//line cmd/internal/parser.go:442
		switch data[p] {
		case 32:
			goto st6
		case 34:
			goto tr13
		case 98:
			goto tr14
		case 104:
			goto tr15
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto st6
		}
		goto st0
tr13:
//line cmd/internal/parser.rl:79

            mark = p
        
//line cmd/internal/parser.rl:109

            valueHolder = nil
        
//line cmd/internal/parser.rl:143
 valueHolder = nil; {stack[top] = 7; top++; goto st53 } 
	goto st7
	st7:
//line NONE:1
ts = 0

		if p++; p == pe {
			goto _test_eof7
		}
	st_case_7:
//line cmd/internal/parser.go:477
		if data[p] == 32 {
			goto tr16
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr16
		}
		goto st0
tr16:
//line cmd/internal/parser.rl:87

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid key"))
            }
            cmd.Key = valueHolder
        
	goto st8
tr49:
//line cmd/internal/parser.rl:113

            valueHolder = decodeBase64(data[mark:p])
        
//line cmd/internal/parser.rl:87

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid key"))
            }
            cmd.Key = valueHolder
        
	goto st8
tr55:
//line cmd/internal/parser.rl:117

            valueHolder = decodeHex(data[mark:p])
        
//line cmd/internal/parser.rl:87

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid key"))
            }
            cmd.Key = valueHolder
        
	goto st8
	st8:
		if p++; p == pe {
			goto _test_eof8
		}
	st_case_8:
//line cmd/internal/parser.go:525
		switch data[p] {
		case 32:
			goto st8
		case 34:
			goto tr18
		case 98:
			goto tr19
		case 104:
			goto tr20
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto st8
		}
		goto st0
tr18:
//line cmd/internal/parser.rl:79

            mark = p
        
//line cmd/internal/parser.rl:109

            valueHolder = nil
        
//line cmd/internal/parser.rl:143
 valueHolder = nil; {stack[top] = 48; top++; goto st53 } 
	goto st48
	st48:
//line NONE:1
ts = 0

		if p++; p == pe {
			goto _test_eof48
		}
	st_case_48:
//line cmd/internal/parser.go:560
		if data[p] == 32 {
			goto tr63
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr63
		}
		goto st0
tr63:
//line cmd/internal/parser.rl:94

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid value"))
            }
            cmd.Value = valueHolder
        
	goto st9
tr64:
//line cmd/internal/parser.rl:105

            cmd.CAS = parseCAS(data[mark:p])
        
	goto st9
tr66:
//line cmd/internal/parser.rl:101

            cmd.Expiry = parseExpiry(data[mark:p])
        
	goto st9
tr67:
//line cmd/internal/parser.rl:113

            valueHolder = decodeBase64(data[mark:p])
        
//line cmd/internal/parser.rl:94

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid value"))
            }
            cmd.Value = valueHolder
        
	goto st9
tr69:
//line cmd/internal/parser.rl:117

            valueHolder = decodeHex(data[mark:p])
        
//line cmd/internal/parser.rl:94

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid value"))
            }
            cmd.Value = valueHolder
        
	goto st9
	st9:
		if p++; p == pe {
			goto _test_eof9
		}
	st_case_9:
//line cmd/internal/parser.go:620
		switch data[p] {
		case 32:
			goto st9
		case 99:
			goto st10
		case 101:
			goto st14
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto st9
		}
		goto st0
	st10:
		if p++; p == pe {
			goto _test_eof10
		}
	st_case_10:
		if data[p] == 97 {
			goto st11
		}
		goto st0
	st11:
		if p++; p == pe {
			goto _test_eof11
		}
	st_case_11:
		if data[p] == 115 {
			goto st12
		}
		goto st0
	st12:
		if p++; p == pe {
			goto _test_eof12
		}
	st_case_12:
		if data[p] == 58 {
			goto st13
		}
		goto st0
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
		if 48 <= data[p] && data[p] <= 57 {
			goto tr27
		}
		goto st0
tr27:
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st49
	st49:
		if p++; p == pe {
			goto _test_eof49
		}
	st_case_49:
//line cmd/internal/parser.go:680
		if data[p] == 32 {
			goto tr64
		}
		switch {
		case data[p] > 13:
			if 48 <= data[p] && data[p] <= 57 {
				goto st49
			}
		case data[p] >= 9:
			goto tr64
		}
		goto st0
	st14:
		if p++; p == pe {
			goto _test_eof14
		}
	st_case_14:
		if data[p] == 120 {
			goto st15
		}
		goto st0
	st15:
		if p++; p == pe {
			goto _test_eof15
		}
	st_case_15:
		if data[p] == 112 {
			goto st16
		}
		goto st0
	st16:
		if p++; p == pe {
			goto _test_eof16
		}
	st_case_16:
		if data[p] == 105 {
			goto st17
		}
		goto st0
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
		if data[p] == 114 {
			goto st18
		}
		goto st0
	st18:
		if p++; p == pe {
			goto _test_eof18
		}
	st_case_18:
		if data[p] == 121 {
			goto st19
		}
		goto st0
	st19:
		if p++; p == pe {
			goto _test_eof19
		}
	st_case_19:
		if data[p] == 58 {
			goto st20
		}
		goto st0
	st20:
		if p++; p == pe {
			goto _test_eof20
		}
	st_case_20:
		if 48 <= data[p] && data[p] <= 57 {
			goto tr34
		}
		goto st0
tr34:
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st21
	st21:
		if p++; p == pe {
			goto _test_eof21
		}
	st_case_21:
//line cmd/internal/parser.go:767
		switch data[p] {
		case 104:
			goto st50
		case 109:
			goto st50
		case 115:
			goto st50
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st21
		}
		goto st0
	st50:
		if p++; p == pe {
			goto _test_eof50
		}
	st_case_50:
		if data[p] == 32 {
			goto tr66
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr66
		}
		goto st0
tr19:
//line cmd/internal/parser.rl:79

            mark = p
        
//line cmd/internal/parser.rl:109

            valueHolder = nil
        
	goto st22
	st22:
		if p++; p == pe {
			goto _test_eof22
		}
	st_case_22:
//line cmd/internal/parser.go:807
		if data[p] == 54 {
			goto st23
		}
		goto st0
	st23:
		if p++; p == pe {
			goto _test_eof23
		}
	st_case_23:
		if data[p] == 52 {
			goto st24
		}
		goto st0
	st24:
		if p++; p == pe {
			goto _test_eof24
		}
	st_case_24:
		if data[p] == 58 {
			goto st25
		}
		goto st0
	st25:
		if p++; p == pe {
			goto _test_eof25
		}
	st_case_25:
		switch data[p] {
		case 43:
			goto tr40
		case 61:
			goto tr40
		}
		switch {
		case data[p] < 65:
			if 47 <= data[p] && data[p] <= 57 {
				goto tr40
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto tr40
			}
		default:
			goto tr40
		}
		goto st0
tr40:
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st51
	st51:
		if p++; p == pe {
			goto _test_eof51
		}
	st_case_51:
//line cmd/internal/parser.go:865
		switch data[p] {
		case 32:
			goto tr67
		case 43:
			goto st51
		case 61:
			goto st51
		}
		switch {
		case data[p] < 47:
			if 9 <= data[p] && data[p] <= 13 {
				goto tr67
			}
		case data[p] > 57:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st51
				}
			case data[p] >= 65:
				goto st51
			}
		default:
			goto st51
		}
		goto st0
tr20:
//line cmd/internal/parser.rl:79

            mark = p
        
//line cmd/internal/parser.rl:109

            valueHolder = nil
        
	goto st26
	st26:
		if p++; p == pe {
			goto _test_eof26
		}
	st_case_26:
//line cmd/internal/parser.go:907
		if data[p] == 101 {
			goto st27
		}
		goto st0
	st27:
		if p++; p == pe {
			goto _test_eof27
		}
	st_case_27:
		if data[p] == 120 {
			goto st28
		}
		goto st0
	st28:
		if p++; p == pe {
			goto _test_eof28
		}
	st_case_28:
		if data[p] == 58 {
			goto st29
		}
		goto st0
	st29:
		if p++; p == pe {
			goto _test_eof29
		}
	st_case_29:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr44
			}
		case data[p] > 70:
			if 97 <= data[p] && data[p] <= 102 {
				goto tr44
			}
		default:
			goto tr44
		}
		goto st0
tr44:
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st52
	st52:
		if p++; p == pe {
			goto _test_eof52
		}
	st_case_52:
//line cmd/internal/parser.go:959
		if data[p] == 32 {
			goto tr69
		}
		switch {
		case data[p] < 48:
			if 9 <= data[p] && data[p] <= 13 {
				goto tr69
			}
		case data[p] > 57:
			switch {
			case data[p] > 70:
				if 97 <= data[p] && data[p] <= 102 {
					goto st52
				}
			case data[p] >= 65:
				goto st52
			}
		default:
			goto st52
		}
		goto st0
tr14:
//line cmd/internal/parser.rl:79

            mark = p
        
//line cmd/internal/parser.rl:109

            valueHolder = nil
        
	goto st30
	st30:
		if p++; p == pe {
			goto _test_eof30
		}
	st_case_30:
//line cmd/internal/parser.go:996
		if data[p] == 54 {
			goto st31
		}
		goto st0
	st31:
		if p++; p == pe {
			goto _test_eof31
		}
	st_case_31:
		if data[p] == 52 {
			goto st32
		}
		goto st0
	st32:
		if p++; p == pe {
			goto _test_eof32
		}
	st_case_32:
		if data[p] == 58 {
			goto st33
		}
		goto st0
	st33:
		if p++; p == pe {
			goto _test_eof33
		}
	st_case_33:
		switch data[p] {
		case 43:
			goto tr48
		case 61:
			goto tr48
		}
		switch {
		case data[p] < 65:
			if 47 <= data[p] && data[p] <= 57 {
				goto tr48
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto tr48
			}
		default:
			goto tr48
		}
		goto st0
tr48:
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st34
	st34:
		if p++; p == pe {
			goto _test_eof34
		}
	st_case_34:
//line cmd/internal/parser.go:1054
		switch data[p] {
		case 32:
			goto tr49
		case 43:
			goto st34
		case 61:
			goto st34
		}
		switch {
		case data[p] < 47:
			if 9 <= data[p] && data[p] <= 13 {
				goto tr49
			}
		case data[p] > 57:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st34
				}
			case data[p] >= 65:
				goto st34
			}
		default:
			goto st34
		}
		goto st0
tr15:
//line cmd/internal/parser.rl:79

            mark = p
        
//line cmd/internal/parser.rl:109

            valueHolder = nil
        
	goto st35
	st35:
		if p++; p == pe {
			goto _test_eof35
		}
	st_case_35:
//line cmd/internal/parser.go:1096
		if data[p] == 101 {
			goto st36
		}
		goto st0
	st36:
		if p++; p == pe {
			goto _test_eof36
		}
	st_case_36:
		if data[p] == 120 {
			goto st37
		}
		goto st0
	st37:
		if p++; p == pe {
			goto _test_eof37
		}
	st_case_37:
		if data[p] == 58 {
			goto st38
		}
		goto st0
	st38:
		if p++; p == pe {
			goto _test_eof38
		}
	st_case_38:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr54
			}
		case data[p] > 70:
			if 97 <= data[p] && data[p] <= 102 {
				goto tr54
			}
		default:
			goto tr54
		}
		goto st0
tr54:
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st39
	st39:
		if p++; p == pe {
			goto _test_eof39
		}
	st_case_39:
//line cmd/internal/parser.go:1148
		if data[p] == 32 {
			goto tr55
		}
		switch {
		case data[p] < 48:
			if 9 <= data[p] && data[p] <= 13 {
				goto tr55
			}
		case data[p] > 57:
			switch {
			case data[p] > 70:
				if 97 <= data[p] && data[p] <= 102 {
					goto st39
				}
			case data[p] >= 65:
				goto st39
			}
		default:
			goto st39
		}
		goto st0
tr3:
//line cmd/internal/parser.rl:75

            cmd = &Command{}
        
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st40
tr7:
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st40
	st40:
		if p++; p == pe {
			goto _test_eof40
		}
	st_case_40:
//line cmd/internal/parser.go:1191
		if data[p] == 101 {
			goto st41
		}
		goto st0
	st41:
		if p++; p == pe {
			goto _test_eof41
		}
	st_case_41:
		if data[p] == 112 {
			goto st42
		}
		goto st0
	st42:
		if p++; p == pe {
			goto _test_eof42
		}
	st_case_42:
		if data[p] == 108 {
			goto st43
		}
		goto st0
	st43:
		if p++; p == pe {
			goto _test_eof43
		}
	st_case_43:
		if data[p] == 97 {
			goto st44
		}
		goto st0
	st44:
		if p++; p == pe {
			goto _test_eof44
		}
	st_case_44:
		if data[p] == 99 {
			goto st45
		}
		goto st0
	st45:
		if p++; p == pe {
			goto _test_eof45
		}
	st_case_45:
		if data[p] == 101 {
			goto st5
		}
		goto st0
tr4:
//line cmd/internal/parser.rl:75

            cmd = &Command{}
        
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st46
tr8:
//line cmd/internal/parser.rl:79

            mark = p
        
	goto st46
	st46:
		if p++; p == pe {
			goto _test_eof46
		}
	st_case_46:
//line cmd/internal/parser.go:1262
		if data[p] == 101 {
			goto st47
		}
		goto st0
	st47:
		if p++; p == pe {
			goto _test_eof47
		}
	st_case_47:
		if data[p] == 116 {
			goto st5
		}
		goto st0
tr71:
//line cmd/internal/parser.rl:140
te = p+1

	goto st53
tr72:
//line cmd/internal/parser.rl:139
te = p+1
{ valueHolder = data[mark+1:p]; {top--; cs = stack[top];goto _again }  }
	goto st53
tr74:
//line cmd/internal/parser.rl:121

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid key"))
            }
        
//line cmd/internal/parser.rl:140
te = p
p--

	goto st53
tr75:
//line cmd/internal/parser.rl:140
te = p
p--

	goto st53
tr76:
//line cmd/internal/parser.rl:138
te = p+1

	goto st53
	st53:
//line NONE:1
ts = 0

		if p++; p == pe {
			goto _test_eof53
		}
	st_case_53:
//line NONE:1
ts = p

//line cmd/internal/parser.go:1320
		switch data[p] {
		case 10:
			goto st0
		case 13:
			goto st0
		case 34:
			goto tr72
		case 92:
			goto st54
		}
		goto tr71
	st54:
		if p++; p == pe {
			goto _test_eof54
		}
	st_case_54:
		if data[p] == 34 {
			goto tr76
		}
		goto tr75
	st_out:
	_test_eof1: cs = 1; goto _test_eof
	_test_eof2: cs = 2; goto _test_eof
	_test_eof3: cs = 3; goto _test_eof
	_test_eof4: cs = 4; goto _test_eof
	_test_eof5: cs = 5; goto _test_eof
	_test_eof6: cs = 6; goto _test_eof
	_test_eof7: cs = 7; goto _test_eof
	_test_eof8: cs = 8; goto _test_eof
	_test_eof48: cs = 48; goto _test_eof
	_test_eof9: cs = 9; goto _test_eof
	_test_eof10: cs = 10; goto _test_eof
	_test_eof11: cs = 11; goto _test_eof
	_test_eof12: cs = 12; goto _test_eof
	_test_eof13: cs = 13; goto _test_eof
	_test_eof49: cs = 49; goto _test_eof
	_test_eof14: cs = 14; goto _test_eof
	_test_eof15: cs = 15; goto _test_eof
	_test_eof16: cs = 16; goto _test_eof
	_test_eof17: cs = 17; goto _test_eof
	_test_eof18: cs = 18; goto _test_eof
	_test_eof19: cs = 19; goto _test_eof
	_test_eof20: cs = 20; goto _test_eof
	_test_eof21: cs = 21; goto _test_eof
	_test_eof50: cs = 50; goto _test_eof
	_test_eof22: cs = 22; goto _test_eof
	_test_eof23: cs = 23; goto _test_eof
	_test_eof24: cs = 24; goto _test_eof
	_test_eof25: cs = 25; goto _test_eof
	_test_eof51: cs = 51; goto _test_eof
	_test_eof26: cs = 26; goto _test_eof
	_test_eof27: cs = 27; goto _test_eof
	_test_eof28: cs = 28; goto _test_eof
	_test_eof29: cs = 29; goto _test_eof
	_test_eof52: cs = 52; goto _test_eof
	_test_eof30: cs = 30; goto _test_eof
	_test_eof31: cs = 31; goto _test_eof
	_test_eof32: cs = 32; goto _test_eof
	_test_eof33: cs = 33; goto _test_eof
	_test_eof34: cs = 34; goto _test_eof
	_test_eof35: cs = 35; goto _test_eof
	_test_eof36: cs = 36; goto _test_eof
	_test_eof37: cs = 37; goto _test_eof
	_test_eof38: cs = 38; goto _test_eof
	_test_eof39: cs = 39; goto _test_eof
	_test_eof40: cs = 40; goto _test_eof
	_test_eof41: cs = 41; goto _test_eof
	_test_eof42: cs = 42; goto _test_eof
	_test_eof43: cs = 43; goto _test_eof
	_test_eof44: cs = 44; goto _test_eof
	_test_eof45: cs = 45; goto _test_eof
	_test_eof46: cs = 46; goto _test_eof
	_test_eof47: cs = 47; goto _test_eof
	_test_eof53: cs = 53; goto _test_eof
	_test_eof54: cs = 54; goto _test_eof

	_test_eof: {}
	if p == eof {
		switch cs {
		case 54:
			goto tr74
		case 50:
//line cmd/internal/parser.rl:101

            cmd.Expiry = parseExpiry(data[mark:p])
        
		case 49:
//line cmd/internal/parser.rl:105

            cmd.CAS = parseCAS(data[mark:p])
        
		case 6, 7, 8, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 53:
//line cmd/internal/parser.rl:121

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid key"))
            }
        
		case 48:
//line cmd/internal/parser.rl:121

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid key"))
            }
        
//line cmd/internal/parser.rl:94

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid value"))
            }
            cmd.Value = valueHolder
        
		case 51:
//line cmd/internal/parser.rl:113

            valueHolder = decodeBase64(data[mark:p])
        
//line cmd/internal/parser.rl:121

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid key"))
            }
        
//line cmd/internal/parser.rl:94

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid value"))
            }
            cmd.Value = valueHolder
        
		case 52:
//line cmd/internal/parser.rl:117

            valueHolder = decodeHex(data[mark:p])
        
//line cmd/internal/parser.rl:121

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid key"))
            }
        
//line cmd/internal/parser.rl:94

            if valueHolder == nil {
                panic(fmt.Errorf("Invalid value"))
            }
            cmd.Value = valueHolder
        
//line cmd/internal/parser.go:1469
		}
	}

	_out: {}
	}

//line cmd/internal/parser.rl:167


    if act < 0 || ts < te{
        return 
    }

    if cs < mnemosyne_first_final {
        err = fmt.Errorf("Invalid command line")
    }

    return 
}


