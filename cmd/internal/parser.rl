package internal

import (
    "fmt"
    "encoding/base64"
    "encoding/hex"
    "time"
    "strconv"
)

%%{
    machine mnemosyne;
    write data;
}%%

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

    %%{
        write init;

        action begin_command {
            cmd = &Command{}
        }

        action mark {
            mark = p
        }

        action add_name_to_command {
            cmd.Name = string(data[mark:p])
        }

        action add_key_to_command {
            if valueHolder == nil {
                panic(fmt.Errorf("Invalid key"))
            }
            cmd.Key = valueHolder
        }

        action add_value_to_command {
            if valueHolder == nil {
                panic(fmt.Errorf("Invalid value"))
            }
            cmd.Value = valueHolder
        }

        action add_expiry_to_command {
            cmd.Expiry = parseExpiry(data[mark:p])
        }

        action add_cas_to_command {
            cmd.CAS = parseCAS(data[mark:p])
        }

        action value_begin {
            valueHolder = nil
        }

        action base64_value_end {
            valueHolder = decodeBase64(data[mark:p])
        }

        action hex_value_end {
            valueHolder = decodeHex(data[mark:p])
        }

        action panic_on_nil_value {
            if valueHolder == nil {
                panic(fmt.Errorf("Invalid key"))
            }
        }

        ws = space+;

        cas_option = "cas:" . digit+ >mark %add_cas_to_command;

        expiry_option = "expiry:" . (digit+ [hms]) >mark %add_expiry_to_command;

        option = expiry_option | cas_option;

        option_list = option (ws option)*;

        inside_string := |*
            '\\"' $/panic_on_nil_value;
            '"' => { valueHolder = data[mark+1:p]; fret;  };
            (any - [\r\n]) $/panic_on_nil_value;
        *|;

        str_value = '"' >mark @{ valueHolder = nil; fcall inside_string; } ;

        hex_value = "hex:" . ( xdigit+ >mark %hex_value_end );

        base64_alphabet = alnum | '+' | '/' | '=';

        base64_value = "b64:" . ( base64_alphabet+ >mark %base64_value_end );

        value = ( base64_value | hex_value | str_value ) >value_begin $/panic_on_nil_value;

        mutation_command = ( 
            ("set" | "add" | "replace") >mark %add_name_to_command )  
            ws 
            ( value >mark %add_key_to_command ) 
            ws  
            ( value >mark %add_value_to_command 
            ( ws option_list )?
        ) ;

        command = ws? mutation_command;

        main := command >begin_command;

        write exec;
    }%%

    if act < 0 || ts < te{
        return 
    }

    if cs < mnemosyne_first_final {
        err = fmt.Errorf("Invalid command line")
    }

    return 
}


