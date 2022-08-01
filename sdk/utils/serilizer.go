package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	ultipa "ultipa-go-sdk/rpc"
	"ultipa-go-sdk/sdk/types"
)

// Convert Bytes to GoLang Type and return to an interface
func ConvertBytesToInterface(bs []byte, t ultipa.PropertyType) interface{} {
	switch t {
	case ultipa.PropertyType_STRING:
		return AsString(bs)
	case ultipa.PropertyType_INT32:
		return AsInt32(bs)
	case ultipa.PropertyType_INT64:
		return AsInt64(bs)
	case ultipa.PropertyType_UINT32:
		return AsUint32(bs)
	case ultipa.PropertyType_UINT64:
		return AsUint64(bs)
	case ultipa.PropertyType_FLOAT:
		return AsFloat32(bs)
	case ultipa.PropertyType_DOUBLE:
		return AsFloat64(bs)
	case ultipa.PropertyType_DATETIME:
		return NewTime(AsUint64(bs))
	case ultipa.PropertyType_TIMESTAMP:
		if len(bs) == 0 {
			return NewTimeStamp(0)
		}
		return NewTimeStamp(int64(AsUint32(bs)))
	case ultipa.PropertyType_UNSET:
		return nil
	default:
		return nil

	}
}

func ConvertInterfaceToBytes(value interface{}) ([]byte, error) {
	v := []byte{}

	switch t := value.(type) {
	case int32:
		v = make([]byte, 4)
		binary.BigEndian.PutUint32(v, uint32(value.(int32)))
	case int:
		v = make([]byte, 4)
		binary.BigEndian.PutUint32(v, uint32(value.(int32)))
	case string:
		v = []byte(value.(string))
	case int64:
		v = make([]byte, 8)
		binary.BigEndian.PutUint64(v, uint64(value.(int64)))
	case uint32:
		v = make([]byte, 4)
		binary.BigEndian.PutUint32(v, uint32(value.(uint32)))
	case uint64:
		v = make([]byte, 8)
		binary.BigEndian.PutUint64(v, uint64(value.(uint64)))
	case float32:
		v = make([]byte, 4)
		binary.BigEndian.PutUint32(v, math.Float32bits(value.(float32)))
	case float64:
		v = make([]byte, 8)
		binary.BigEndian.PutUint64(v, math.Float64bits(value.(float64)))
	default:
		return nil, errors.New(fmt.Sprint("not supported ultipa type : ", t))
	}

	return v, nil
}

func readBytes(value []byte, out interface{}) {
	bs := make([]byte, len(value))
	copy(bs, value)
	buff := bytes.NewBuffer(bs)
	binary.Read(buff, binary.BigEndian, out)
}

func AsInt32(value []byte) int32 {
	var out int32
	readBytes(value, &out)
	return out
}

func AsInt64(value []byte) int64 {
	var out int64
	readBytes(value, &out)
	return out
}

func AsUint32(value []byte) uint32 {
	var out uint32
	readBytes(value, &out)
	return out
}

func AsUint64(value []byte) uint64 {
	var out uint64
	readBytes(value, &out)
	return out
}

func AsFloat32(value []byte) float32 {
	var out float32
	readBytes(value, &out)
	return out
}

func AsFloat64(value []byte) float64 {
	var out float64
	readBytes(value, &out)
	return out
}

func AsString(value []byte) string {
	return string(value)
}

func AsBool(value []byte) bool {
	var out uint16
	readBytes(value, &out)
	if out == 1 {
		return true
	} else {
		return false
	}
}

func GetDefaultNilString(t ultipa.PropertyType) string {

	switch t {
	case ultipa.PropertyType_INT32:
		fallthrough
	case ultipa.PropertyType_INT64:
		fallthrough
	case ultipa.PropertyType_UINT32:
		fallthrough
	case ultipa.PropertyType_UINT64:
		fallthrough

	case ultipa.PropertyType_FLOAT:
		fallthrough
	case ultipa.PropertyType_DOUBLE:
		return "0"
	case ultipa.PropertyType_DATETIME:
		return "1970-01-01"
	case ultipa.PropertyType_TIMESTAMP:
		return "1970-01-01"
	default:
		return ""
	}

}

func StringAsInterface(str string, t ultipa.PropertyType) (interface{}, error) {

	str = strings.Trim(str, " ")

	if str == "" {
		str = GetDefaultNilString(t)
	}

	switch t {
	case ultipa.PropertyType_INT32:
		v, err := strconv.ParseInt(str, 10, 32)
		if err != nil {
			return nil, err
		}
		return int32(v), err
	case ultipa.PropertyType_INT64:
		return strconv.ParseInt(str, 10, 64)
	case ultipa.PropertyType_UINT32:
		v, err := strconv.ParseUint(str, 10, 32)
		if err != nil {
			return nil, err
		}
		return uint32(v), err
	case ultipa.PropertyType_UINT64:
		return strconv.ParseUint(str, 10, 64)
	case ultipa.PropertyType_FLOAT:
		v, err := strconv.ParseFloat(str, 32)
		if err != nil {
			return nil, err
		}
		return float32(v), err
	case ultipa.PropertyType_DOUBLE:
		v, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return nil, err
		}
		return float64(v), err
	case ultipa.PropertyType_DATETIME:
		v, err := NewTimeFromString(str)
		if err != nil {
			return nil, err
		}
		return v.Datetime, err
	case ultipa.PropertyType_TIMESTAMP:
		v, err := NewTimeFromString(str)
		if err != nil {
			return nil, err
		}
		return v.GetTimeStamp(), err
	default:
		return str, nil
	}

	return nil, nil
}

func StringAsUUID(str string) (types.UUID, error) {
	str = strings.Trim(str, " ")
	v, err := strconv.ParseUint(str, 10, 64)
	return v, err
}
