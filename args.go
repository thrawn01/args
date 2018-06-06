package args

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	DefaultTerminator  string = "--"
	DefaultOptionGroup string = ""
)

var DefaultLogger *NullLogger = &NullLogger{}

// We only need part of the standard logging functions
type StdLogger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}

type NullLogger struct{}

func (nl *NullLogger) Print(...interface{})          {}
func (nl *NullLogger) Printf(string, ...interface{}) {}
func (nl *NullLogger) Println(...interface{})        {}

// ***********************************************
// Public Word Formatting Functions
// ***********************************************

func castString(name string, dest interface{}, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return "", nil
	}

	// If value is not an string
	if reflect.TypeOf(value).Kind() != reflect.String {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a String", name, value))
	}
	return value, nil
}

func castInt(name string, dest interface{}, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return 0, nil
	}

	// If it's already an integer of some sort
	kind := reflect.TypeOf(value).Kind()
	switch kind {
	case reflect.Int:
		return value, nil
	case reflect.Int8:
		return int(value.(int8)), nil
	case reflect.Int16:
		return int(value.(int16)), nil
	case reflect.Int32:
		return int(value.(int32)), nil
	case reflect.Int64:
		return int(value.(int64)), nil
	case reflect.Uint8:
		return int(value.(uint8)), nil
	case reflect.Uint16:
		return int(value.(uint16)), nil
	case reflect.Uint32:
		return int(value.(uint32)), nil
	case reflect.Uint64:
		return int(value.(uint64)), nil
	}
	// If it's not an integer, it better be a string that we can cast
	if kind != reflect.String {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a Integer or Castable string", name, value))
	}
	strValue := value.(string)

	intValue, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not an Integer", name, strValue))
	}
	return int(intValue), nil
}

func castBool(name string, dest interface{}, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return false, nil
	}

	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Bool {
		return value, nil
	}
	// If it's not a boolean, it better be a string that we can cast
	if kind != reflect.String {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a Boolean or Castable string", name, value))
	}
	strValue := value.(string)

	boolValue, err := strconv.ParseBool(strValue)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a Boolean", name, strValue))
	}
	return bool(boolValue), nil
}

func castStringSlice(name string, dest interface{}, value interface{}) (interface{}, error) {
	// If our destination is nil, init a new slice
	if dest == nil {
		dest = make([]string, 0)
	}

	// If value is nil, return the type default
	if value == nil {
		return dest, nil
	}

	// value could already be a slice
	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Slice {
		sliceKind := reflect.TypeOf(value).Elem().Kind()
		// Is already a []string
		if sliceKind == reflect.String {
			return append(dest.([]string), value.([]string)...), nil
		}
		return dest, errors.New(fmt.Sprintf("Invalid slice type for '%s' - '%s'  not a String",
			name, value))
	}

	// or it could be a string
	if kind != reflect.String {
		return dest, errors.New(fmt.Sprintf("Invalid slice type for '%s' - '%s' is not a "+
			"[]string or parsable comma delimited string", name, value))
	}

	// Assume the value must be a parsable string
	return append(dest.([]string), StringToSlice(value.(string), strings.TrimSpace)...), nil
}

func mergeStringMap(src, dest map[string]string) map[string]string {
	for key, value := range src {
		dest[key] = value
	}
	return dest
}

func isMapString(value interface{}) bool {
	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Map {
		kind := reflect.TypeOf(value).Elem().Kind()
		// Is already a string
		if kind == reflect.String {
			return true
		}
	}
	return false
}

func castStringMap(name string, dest interface{}, value interface{}) (interface{}, error) {
	// If our destination is nil, init a new slice
	if dest == nil {
		dest = make(map[string]string, 0)
	}

	// Don't attempt to cast a nil value
	if value == nil {
		return nil, nil
	}

	// could already be a map[string]string
	if isMapString(value) {
		return mergeStringMap(dest.(map[string]string), value.(map[string]string)), nil
	}

	// value should be a string
	if reflect.TypeOf(value).Kind() != reflect.String {
		return dest, errors.New(fmt.Sprintf("Invalid map type for '%s' - '%s' is not a "+
			"map[string]string or parsable key=value string", name, reflect.TypeOf(value)))
	}

	// Assume the value is a parsable string
	strValue := value.(string)
	// Parse the string
	result, err := StringToMap(strValue)
	if err != nil {
		return dest, errors.New(fmt.Sprintf("Invalid map type for '%s' - %s", name, err))
	}

	return mergeStringMap(dest.(map[string]string), result), nil
}

func containsString(needle string, haystack []string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}

func copyStringSlice(src []string) (dest []string) {
	dest = make([]string, len(src))
	for idx, value := range src {
		dest[idx] = value
	}
	return
}

func JSONToMap(value string) (map[string]string, error) {
	result := make(map[string]string)
	err := json.Unmarshal([]byte(value), &result)
	if err != nil {
		return result, errors.New(fmt.Sprintf("JSON map decoding for '%s' failed with '%s'; "+
			`JSON map values should be in form '{"key":"value", "foo":"bar"}'`, value, err))
	}
	return result, nil
}

