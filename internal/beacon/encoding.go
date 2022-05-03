package beacon

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
)

func Marshal(obj interface{}) ([]byte, error) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("not a struct")
	}

	raw, err := marshalImpl(val)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func marshalImpl(v reflect.Value) (interface{}, error) {
	typ := v.Type()
	if isByteArray(typ) {
		// [n]byte
		return "0x" + hex.EncodeToString(convertArrayToBytes(v).Bytes()), nil
	}
	if isByteSlice(typ) {
		// []byte
		return "0x" + hex.EncodeToString(v.Bytes()), nil
	}

	switch v.Kind() {
	case reflect.Ptr:
		return marshalImpl(v.Elem())

	case reflect.Array, reflect.Slice:
		out := []interface{}{}
		for i := 0; i < v.Len(); i++ {
			out = append(out, v.Index(i))
		}
		return out, nil

	case reflect.Struct:
		out := map[string]interface{}{}
		for i := 0; i < v.NumField(); i++ {
			f := typ.Field(i)

			val, err := marshalImpl(v.Field(i))
			if err != nil {
				return nil, err
			}

			tagValue := f.Tag.Get("json")
			if tagValue == "-" {
				continue
			}

			name := f.Name
			if tagValue != "" {
				name = tagValue
			}
			out[name] = val
		}
		return out, nil

	case reflect.String:
		return v.String(), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.Itoa(int(v.Uint())), nil
	}

	return nil, nil
}

func convertArrayToBytes(value reflect.Value) reflect.Value {
	slice := reflect.MakeSlice(reflect.TypeOf([]byte{}), value.Len(), value.Len())
	reflect.Copy(slice, value)
	return slice
}

func Unmarshal(data []byte, obj interface{}) error {
	var out1 interface{}
	if err := json.Unmarshal(data, &out1); err != nil {
		return err
	}
	dc := &mapstructure.DecoderConfig{
		Result:           obj,
		WeaklyTypedInput: true,
		DecodeHook:       customWeb3Hook,
		TagName:          "json",
	}
	ms, err := mapstructure.NewDecoder(dc)
	if err != nil {
		return err
	}
	if err = ms.Decode(out1); err != nil {
		return err
	}
	return nil
}

func isByteArray(t reflect.Type) bool {
	return t.Kind() == reflect.Array && t.Elem().Kind() == reflect.Uint8
}

func isByteSlice(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8
}

func customWeb3Hook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}

	if isByteArray(t) {
		raw := data.(string)
		if !strings.HasPrefix(raw, "0x") {
			return nil, fmt.Errorf("0x prefix not found")
		}
		elem, err := hex.DecodeString(raw[2:])
		if err != nil {
			return nil, err
		}

		// [n]byte
		if t.Len() != len(elem) {
			return nil, fmt.Errorf("incorrect array length: %d %d", t.Len(), len(elem))
		}

		v := reflect.New(t)
		reflect.Copy(v.Elem(), reflect.ValueOf(elem))
		return v.Interface(), nil
	}

	if isByteSlice(t) {
		// []byte
		raw := data.(string)
		if !strings.HasPrefix(raw, "0x") {
			return nil, fmt.Errorf("0x prefix not found")
		}
		elem, err := hex.DecodeString(raw[2:])
		if err != nil {
			return nil, err
		}
		return elem, nil
	}

	return data, nil
}
