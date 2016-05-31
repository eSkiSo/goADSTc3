package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"time"
)

type Pair struct {
	Key   string
	Value int
}
type PairList []Pair

func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }

//func (dt *ADSSymbol) parse(offset uint32, data []byte) { /*{{{*/
func (dt *ADSSymbol) parse(data []byte, offset int) { /*{{{*/
	start := offset
	stop := start + int(dt.Length)

	if dt.Childs != nil {
		for _, value := range dt.Childs {
			value.parse(data[offset:stop], int(value.Offset))
		}
	}
	// for i := range dt.Childs {

	// 	dt.Childs[i].Self.parse(data)
	// }

	if len(dt.Childs) == 0 {
		var newValue = "nil"

		if len(data) < int(dt.Length) {
			fmt.Errorf("Incoming data is to small, !0<%d<%d<%d", start, stop, len(data))
			return
		}

		switch dt.DataType {
		case "BOOL":
			if stop-start != 1 {
				return
			}
			if data[start:stop][0] > 0 {
				newValue = "True"
			} else {
				newValue = "False"
			}
		case "BYTE", "USINT": // Unsigned Short INT 0 to 255
			if stop-start != 1 {
				return
			}
			buf := bytes.NewBuffer(data[start:stop])
			var i uint8
			binary.Read(buf, binary.LittleEndian, &i)
			newValue = strconv.FormatInt(int64(i), 10)
		case "SINT": // Short INT -128 to 127
			if stop-start != 1 {
				return
			}
			buf := bytes.NewBuffer(data[start:stop])
			var i int8
			binary.Read(buf, binary.LittleEndian, &i)
			newValue = strconv.FormatInt(int64(i), 10)
		case "UINT", "WORD":
			if stop-start != 2 {
				return
			}
			i := binary.LittleEndian.Uint16(data[start:stop])
			newValue = strconv.FormatUint(uint64(i), 10)
		case "UDINT", "DWORD":
			if stop-start != 4 {
				return
			}
			i := binary.LittleEndian.Uint32(data[start:stop])
			newValue = strconv.FormatUint(uint64(i), 10)
		case "INT":
			if stop-start != 2 {
				return
			}
			buf := bytes.NewBuffer(data)
			var i int16
			binary.Read(buf, binary.LittleEndian, &i)
			i = int16(binary.LittleEndian.Uint16(data[start:stop]))
			newValue = strconv.FormatInt(int64(i), 10)
		case "DINT":
			if stop-start != 4 {
				return
			}
			buf := bytes.NewBuffer(data[start:stop])
			var i int32
			binary.Read(buf, binary.LittleEndian, &i)
			newValue = strconv.FormatInt(int64(i), 10)
		case "REAL":
			if stop-start != 4 {
				return
			}
			i := binary.LittleEndian.Uint32(data[start:stop])
			f := math.Float32frombits(i)
			newValue = strconv.FormatFloat(float64(f), 'f', -1, 32)
		case "LREAL":
			if stop-start != 8 {
				return
			}
			i := binary.LittleEndian.Uint64(data[start:stop])
			f := math.Float64frombits(i)
			newValue = strconv.FormatFloat(f, 'f', -1, 64)
		case "STRING":
			trimmedBytes := bytes.TrimSpace(data[start:stop])
			secondIndex := bytes.IndexByte(trimmedBytes, byte(0))
			newValue = string(trimmedBytes[:(secondIndex)])
		case "TIME":
			if stop-start != 4 {
				return
			}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Millisecond))-int64(time.Hour))

			newValue = t.Truncate(time.Millisecond).Format("15:04:05.999999999")
		case "TOD":
			if stop-start != 4 {
				return
			}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Millisecond))-int64(time.Hour))

			newValue = t.Truncate(time.Millisecond).Format("15:04")
		case "DATE":
			if stop-start != 4 {
				return
			}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Second)))

			newValue = t.Truncate(time.Millisecond).Format("2006-01-02")
		case "DT":
			if stop-start != 4 {
				return
			}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Second))-int64(time.Hour))

			newValue = t.Truncate(time.Millisecond).Format("2006-01-02 15:04:05")
		default:
			newValue = "nil"
		}

		if strcmp(dt.Value, newValue) != 0 {
			dt.Value = newValue
			dt.Changed = dt.Valid
			fmt.Println(dt.FullName, dt.Value)
		}

		dt.Valid = true
	}
}

func (symbol *ADSSymbol) Write(value string, offset int) (err error) { /*{{{*/
	// log.Warn("Write (", symbol.Area, ":", symbol.Offset, "): ", symbol.Name)

	if len(symbol.Childs) != 0 {
		err = fmt.Errorf("Cannot write to a whole struct at once!")

		return
	}

	buf := bytes.NewBuffer([]byte{})

	switch symbol.DataType {
	case "BOOL":
		v, e := strconv.ParseBool(value)
		if e != nil {
			return e
		}

		if v {
			buf.Write([]byte{1})
		} else {
			buf.Write([]byte{0})
		}
	case "BYTE", "USINT": // Unsigned Short INT 0 to 255
		v, e := strconv.ParseUint(value, 10, 8)
		if e != nil {
			return e
		}

		v8 := uint8(v)
		binary.Write(buf, binary.LittleEndian, &v8)
	case "UINT", "WORD":
		v, e := strconv.ParseUint(value, 10, 16)
		if e != nil {
			return e
		}

		v16 := uint16(v)
		binary.Write(buf, binary.LittleEndian, &v16)
	case "UDINT", "DWORD":
		v, e := strconv.ParseUint(value, 10, 32)
		if e != nil {
			return e
		}

		v32 := uint32(v)
		binary.Write(buf, binary.LittleEndian, &v32)

	case "SINT": // Short INT -128 to 127
		v, e := strconv.ParseInt(value, 10, 8)
		if e != nil {
			return e
		}

		v8 := int8(v)
		binary.Write(buf, binary.LittleEndian, &v8)
	case "INT":
		v, e := strconv.ParseInt(value, 10, 16)
		if e != nil {
			return e
		}

		v16 := int16(v)
		binary.Write(buf, binary.LittleEndian, &v16)
	case "DINT":
		v, e := strconv.ParseInt(value, 10, 32)
		if e != nil {
			return e
		}

		v32 := int32(v)
		binary.Write(buf, binary.LittleEndian, &v32)

	case "REAL":
		v, e := strconv.ParseFloat(value, 32)
		if e != nil {
			return e
		}

		v32 := math.Float32bits(float32(v))
		binary.Write(buf, binary.LittleEndian, &v32)
	case "LREAL":
		v, e := strconv.ParseFloat(value, 64)
		if e != nil {
			return e
		}

		v64 := math.Float64bits(v)
		binary.Write(buf, binary.LittleEndian, &v64)
	case "STRING":
		buf.Write([]byte(value))
	/*case "TIME":
		if stop-start != 4 {return}
		i := binary.LittleEndian.Uint32(data[start:stop])
		t := time.Unix(0, int64(uint64(i)*uint64(time.Millisecond))-int64(time.Hour) )

		newValue = t.Truncate(time.Millisecond).Format("15:04:05.999999999")
	case "TOD":
		if stop-start != 4 {return}
		i := binary.LittleEndian.Uint32(data[start:stop])
		t := time.Unix(0, int64(uint64(i)*uint64(time.Millisecond))-int64(time.Hour) )

		newValue = t.Truncate(time.Millisecond).Format("15:04")
	case "DATE":
		if stop-start != 4 {return}
		i := binary.LittleEndian.Uint32(data[start:stop])
		t := time.Unix(0, int64(uint64(i)*uint64(time.Second)) )

		newValue = t.Truncate(time.Millisecond).Format("2006-01-02")
	case "DT":
		if stop-start != 4 {return}
		i := binary.LittleEndian.Uint32(data[start:stop])
		t := time.Unix(0, int64(uint64(i)*uint64(time.Second))-int64(time.Hour) )

		newValue = t.Truncate(time.Millisecond).Format("2006-01-02 15:04:05")*/
	default:
		err = fmt.Errorf("Datatype '%s' write is not implemented yet!", symbol.DataType)

		return
	}

	// set
	//symbol.Self.conn.Write(symbol.Area, symbol.Offset, buf.Bytes())

	return nil

}

func strcmp(a, b string) int {
	min := len(b)
	if len(a) < len(b) {
		min = len(a)
	}
	diff := 0
	for i := 0; i < min && diff == 0; i++ {
		diff = int(a[i]) - int(b[i])
	}
	if diff == 0 {
		diff = len(a) - len(b)
	}
	return diff
}
