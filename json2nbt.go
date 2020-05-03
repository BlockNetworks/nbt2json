package nbt2json

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/ghodss/yaml"
)

// Yaml2Nbt converts JSON byte array to uncompressed NBT byte array
func Yaml2Nbt(b []byte) ([]byte, error) {
	myJson, err := yaml.YAMLToJSON(b)
	if err != nil {
		return nil, JsonParseError{"Error converting YAML to JSON", err}
	}
	nbtOut, err := Json2Nbt(myJson)
	if err != nil {
		return nbtOut, err
	}
	return nbtOut, nil
}

// Json2Nbt converts JSON byte array to uncompressed NBT byte array
func Json2Nbt(b []byte) ([]byte, error) {
	nbtOut := new(bytes.Buffer)
	var nbtJsonData NbtJson
	var nbtTag interface{}
	var nbtArray []interface{}
	var err error
	d := json.NewDecoder(bytes.NewBuffer(b))
	d.UseNumber()

	err = d.Decode(&nbtJsonData)
	if err != nil {
		return nil, JsonParseError{"Error parsing JSON input. Is input JSON-formatted?", err}
	}
	temp, err := json.Marshal(nbtJsonData.Nbt)
	if err != nil {
		return nil, JsonParseError{"Error marshalling nbt: json.RawMessage", err}
	}
	d2 := json.NewDecoder(bytes.NewBuffer(temp))
	d2.UseNumber()
	err = d2.Decode(&nbtArray)
	if err != nil {
		return nil, JsonParseError{"Error unmarshalling nbt: value", err}
	}
	if len(nbtArray) == 0 {
		return nil, JsonParseError{"JSON input has no top-level value named nbt. JSON-encoded nbt data should be in an array { \"nbt\": [ <HERE> ] }", nil}
	}
	for _, nbtTag = range nbtArray {
		err = writeTag(nbtOut, nbtTag)
		if err != nil {
			return nil, err
		}
	}

	return nbtOut.Bytes(), nil
}

func writeTag(w io.Writer, myMap interface{}) error {
	var err error
	// TODO: This is panic-exiting when passed a string or null tagType instead of returning error
	if m, ok := myMap.(map[string]interface{}); ok {
		if tagType, err := m["tagType"].(json.Number).Int64(); err == nil {
			if tagType == 0 {
				// not expecting a 0 tag, but if it occurs just ignore it
				return nil
			}
			err = binary.Write(w, byteOrder, byte(tagType))
			if err != nil {
				return JsonParseError{"Error writing tagType" + string(byte(tagType)), err}
			}
			if name, ok := m["name"].(string); ok {
				err = binary.Write(w, byteOrder, int16(len(name)))
				if err != nil {
					return JsonParseError{"Error writing name length", err}
				}
				err = binary.Write(w, byteOrder, []byte(name))
				if err != nil {
					return JsonParseError{"Error converting name", err}
				}
			} else {
				return JsonParseError{fmt.Sprintf("name field '%v' not a string", m["name"]), err}
			}
			err = writePayload(w, m, tagType)
			if err != nil {
				return err
			}

		} else {
			return JsonParseError{fmt.Sprintf("tagType '%v' is not an integer", m["tagType"]), err}
		}
	} else {
		return JsonParseError{"writeTag: myMap is not map[string]interface{}", err}
	}
	return err
}

func writePayload(w io.Writer, m map[string]interface{}, tagType int64) error {
	var err error

	switch tagType {
	case 1:
		if i, err := m["value"].(json.Number).Int64(); err == nil {
			if i < math.MinInt8 || i > math.MaxInt8 {
				return JsonParseError{fmt.Sprintf("%d is out of range for tag 1 - Byte", i), nil}
			}
			err = binary.Write(w, byteOrder, int8(i))
			if err != nil {
				return JsonParseError{"Error writing byte payload", err}
			}
		} else {
			return JsonParseError{fmt.Sprintf("Tag 1 Byte value field '%v' not an integer", m["value"]), err}
		}
	case 2:
		if i, err := m["value"].(json.Number).Int64(); err == nil {
			if i < math.MinInt16 || i > math.MaxInt16 {
				return JsonParseError{fmt.Sprintf("%d is out of range for tag 2 - Short", i), nil}
			}
			err = binary.Write(w, byteOrder, int16(i))
			if err != nil {
				return JsonParseError{"Error writing short payload", err}
			}
		} else {
			return JsonParseError{fmt.Sprintf("Tag 2 Short value field '%v' not an integer", m["value"]), err}
		}
	case 3:
		if i, err := m["value"].(json.Number).Int64(); err == nil {
			if i < math.MinInt32 || i > math.MaxInt32 {
				return JsonParseError{fmt.Sprintf("%d is out of range for tag 3 - Int", i), nil}
			}
			err = binary.Write(w, byteOrder, int32(i))
			if err != nil {
				return JsonParseError{"Error writing int32 payload", err}
			}
		} else {
			return JsonParseError{fmt.Sprintf("Tag 3 Int value field '%v' not an integer", m["value"]), err}
		}
	case 4:
		if int64Map, ok := m["value"].(map[string]interface{}); ok {
			var nbtLong NbtLong
			var vl, vm int64
			if vl, err = int64Map["valueLeast"].(json.Number).Int64(); err != nil {
				return JsonParseError{fmt.Sprintf("Error reading valueLeast of '%v'", int64Map["valueLeast"]), nil}
			}
			nbtLong.ValueLeast = uint32(vl)
			if vm, err = int64Map["valueMost"].(json.Number).Int64(); err != nil {
				return JsonParseError{fmt.Sprintf("Error reading valueMost of '%v'", int64Map["valueLeast"]), nil}
			}
			nbtLong.ValueMost = uint32(vm)
			err = binary.Write(w, byteOrder, int64(intPairToLong(nbtLong)))
			if err != nil {
				return JsonParseError{"Error writing int64 payload", err}
			}
		} else {
			return JsonParseError{fmt.Sprintf("Tag 4 Long value field '%v' not an object", m["value"]), err}
		}
	case 5:
		if f, err := m["value"].(json.Number).Float64(); err == nil {
			if f != 0 && (math.Abs(f) < math.SmallestNonzeroFloat32 || math.Abs(f) > math.MaxFloat32) {
				return JsonParseError{fmt.Sprintf("%g is out of range for tag 5 - Float", f), nil}
			}
			err = binary.Write(w, byteOrder, float32(f))
			if err != nil {
				return JsonParseError{"Error writing float32 payload", err}
			}
		} else {
			return JsonParseError{fmt.Sprintf("Tag 5 Float value field '%v' not a number", m["value"]), err}
		}
	case 6:
		if f, err := m["value"].(json.Number).Float64(); err == nil {
			err = binary.Write(w, byteOrder, f)
			if err != nil {
				return JsonParseError{"Error writing float64 payload", err}
			}
		} else { // TODO: not tested with json.Number
			// return JsonParseError{fmt.Sprintf("Tag 6 Double value field '%v' not a number", m["value"]), err}
			f = math.NaN()
			err = binary.Write(w, byteOrder, f)
			if err != nil {
				return JsonParseError{"Error writing float64 payload", err}
			}

		}
	case 7:
		if values, ok := m["value"].([]interface{}); ok {
			err = binary.Write(w, byteOrder, int32(len(values)))
			if err != nil {
				return JsonParseError{"Error writing byte array length", err}
			}
			for _, value := range values {
				if i, err := value.(json.Number).Int64(); err == nil {
					if i < math.MinInt8 || i > math.MaxInt8 {
						return JsonParseError{fmt.Sprintf("%d is out of range for Byte in tag 7 - Byte Array", i), nil}
					}
					err = binary.Write(w, byteOrder, int8(i))
					if err != nil {
						return JsonParseError{"Error writing element of byte array", err}
					}
				} else {
					return JsonParseError{fmt.Sprintf("Tag 7 Byte Array element value field '%v' not an integer", m["value"]), err}
				}
			}
		} else {
			return JsonParseError{fmt.Sprintf("Tag 7 Byte Array element value field '%v' not an array", m["value"]), err}
		}
	case 8:
		if s, ok := m["value"].(string); ok {
			err = binary.Write(w, byteOrder, int16(len([]byte(s))))
			if err != nil {
				return JsonParseError{"Error writing string length", err}
			}
			err = binary.Write(w, byteOrder, []byte(s))
			if err != nil {
				return JsonParseError{"Error writing string payload", err}
			}
		} else {
			return JsonParseError{fmt.Sprintf("Tag 8 String value field '%v' not a string", m["value"]), err}
		}
	case 9:
		// important: tagListType needs to be in scope to be passed to writePayload
		// := were keeping it in a lower scope and zeroing it out.
		var tagListType int64
		if listMap, ok := m["value"].(map[string]interface{}); ok {
			if tagListType, err = listMap["tagListType"].(json.Number).Int64(); err == nil {
				err = binary.Write(w, byteOrder, byte(tagListType))
				if err != nil {
					return JsonParseError{"While writing tag 9 list type", err}
				}
			}
			if values, ok := listMap["list"].([]interface{}); ok {
				err = binary.Write(w, byteOrder, int32(len(values)))
				if err != nil {
					return JsonParseError{"While writing tag 9 list size", err}
				}
				for _, value := range values {
					fakeTag := make(map[string]interface{})
					fakeTag["value"] = value
					err = writePayload(w, fakeTag, tagListType)
					if err != nil {
						return JsonParseError{"While writing tag 9 list of type " + strconv.Itoa(int(tagListType)), err}
					}
				}
			} else if listMap["list"] == nil {
				// NBT lists can be null / nil and therefore aren't represented as an array in JSON
				err = binary.Write(w, byteOrder, int32(0))
				if err != nil {
					return JsonParseError{"While writing tag 9 list null size", err}
				}
				return nil
			} else {
				return JsonParseError{fmt.Sprintf("Tag 9 List's value field '%v' not an array or null", listMap["list"]), err}
			}

		} else {
			return JsonParseError{fmt.Sprintf("Tag 9 List value field '%v' not an object", m["value"]), err}
		}
	case 10:
		if values, ok := m["value"].([]interface{}); ok {
			for _, value := range values {
				err = writeTag(w, value)
				if err != nil {
					return JsonParseError{"While writing Compound tags", err}
				}
			}
			// write the end tag which is just a single byte 0
			err = binary.Write(w, byteOrder, byte(0))
			if err != nil {
				return JsonParseError{"Writing End tag", err}
			}
		} else {
			return JsonParseError{fmt.Sprintf("Tag 10 Compound value field '%v' not an array", m["value"]), err}
		}
	case 11:
		if values, ok := m["value"].([]interface{}); ok {
			err = binary.Write(w, byteOrder, int32(len(values)))
			if err != nil {
				return JsonParseError{"Error writing int32 array length", err}
			}
			for _, value := range values {
				if i, err := value.(json.Number).Int64(); err == nil {
					if i < math.MinInt32 || i > math.MaxInt32 {
						return JsonParseError{fmt.Sprintf("%d is out of range for Int in tag 11 - Int Array", i), nil}
					}
					err = binary.Write(w, byteOrder, int32(i))
					if err != nil {
						return JsonParseError{"Error writing element of int32 array", err}
					}
				} else {
					return JsonParseError{fmt.Sprintf("Tag 11 Int Array element value field '%v' not an integer", value), err}
				}
			}
		} else {
			return JsonParseError{fmt.Sprintf("Tag Int Array value field '%v' not an array", m["value"]), err}
		}
	case 12:
		if values, ok := m["value"].([]interface{}); ok {
			err = binary.Write(w, byteOrder, int64(len(values)))
			if err != nil {
				return JsonParseError{"Error writing int64 array length", err}
			}
			for _, value := range values {
				if int64Map, ok := value.(map[string]interface{}); ok {
					var nbtLong NbtLong
					var vl, vm int64
					if vl, err = int64Map["valueLeast"].(json.Number).Int64(); err != nil {
						return JsonParseError{fmt.Sprintf("Error reading valueLeast of '%v'", int64Map["valueLeast"]), nil}
					}
					nbtLong.ValueLeast = uint32(vl)
					if vm, err = int64Map["valueMost"].(json.Number).Int64(); err != nil {
						return JsonParseError{fmt.Sprintf("Error reading valueMost of '%v'", int64Map["valueMost"]), nil}
					}
					nbtLong.ValueMost = uint32(vm)
					// if i, err := value.(json.Number).Int64(); err == nil {
					err = binary.Write(w, byteOrder, int64(intPairToLong(nbtLong)))
					if err != nil {
						return JsonParseError{"Error writing element of int64 array", err}
					}
				} else {
					return JsonParseError{fmt.Sprintf("Tag Long Array element value field '%v' not an object", value), err}
				}
			}
		} else {
			return JsonParseError{fmt.Sprintf("Tag 12 Long Array element value field '%v' not an array", m["value"]), err}
		}
	default:
		return JsonParseError{fmt.Sprintf("tagType '%v' is not recognized", tagType), err}
	}
	return err
}
