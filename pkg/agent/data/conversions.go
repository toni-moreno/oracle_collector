package data

import (
	"strconv"
	"strings"
)

func convert2Int64(value interface{}) int64 {
	var val int64
	// revisar esta asignación
	switch value := value.(type) { // shadow
	case float64:
		val = int64(value)
	case float32:
		val = int64(value)
	case int:
		val = int64(value)
	case int8:
		val = int64(value)
	case int16:
		val = int64(value)
	case int32:
		val = int64(value)
	case int64:
		val = int64(value)
	case uint:
		val = int64(value)
	case uint8:
		val = int64(value)
	case uint16:
		val = int64(value)
	case uint32:
		val = int64(value)
	case uint64:
		val = int64(value)
	case string:
		// for testing and other apps - numbers may appear as strings
		var err error
		if val, err = strconv.ParseInt(value, 10, 64); err != nil {
			return val
		}
	default:
		return 0
	}
	return val
}

func convert2Float(value interface{}) float64 {
	var val float64
	// revisar esta asignación
	switch value := value.(type) { // shadow
	case float64:
		val = value
	case float32:
		val = float64(value)
	case int:
		val = float64(value)
	case int8:
		val = float64(value)
	case int16:
		val = float64(value)
	case int32:
		val = float64(value)
	case int64:
		val = float64(value)
	case uint:
		val = float64(value)
	case uint8:
		val = float64(value)
	case uint16:
		val = float64(value)
	case uint32:
		val = float64(value)
	case uint64:
		val = float64(value)
	case string:
		// for testing and other apps - numbers may appear as strings
		var err error
		if val, err = strconv.ParseFloat(value, 64); err != nil {
			return val
		}
	default:
		return 0.0
	}
	return val
}

// PduVal2UInt64 transform data to Uint64
func convert2String(value interface{}) string {
	var val string
	// revisar esta asignación
	switch value := value.(type) { // shadow
	case float64:
		val = strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		val = strconv.FormatFloat(float64(value), 'f', -1, 32)
	case int:
		val = strconv.Itoa(int(value))
	case int8:
		val = strconv.FormatInt(int64(value), 10)
	case int16:
		val = strconv.FormatInt(int64(value), 10)
	case int32:
		val = strconv.FormatInt(int64(value), 10)
	case int64:
		val = strconv.FormatInt(int64(value), 10)
	case uint:
		val = strconv.FormatUint(uint64(value), 10)
	case uint8:
		val = strconv.FormatUint(uint64(value), 10)
	case uint16:
		val = strconv.FormatUint(uint64(value), 10)
	case uint32:
		val = strconv.FormatUint(uint64(value), 10)
	case uint64:
		val = strconv.FormatUint(uint64(value), 10)
	case string:
		val = strings.TrimSpace(value)
	default:
		return ""
	}
	return val
}

func convert2Bool(value interface{}) bool {
	var val bool = false
	// revisar esta asignación
	switch value := value.(type) { // shadow
	case int:
		if value > 0 {
			val = true
		}
	case int8:
		if value > 0 {
			val = true
		}
	case int16:
		if value > 0 {
			val = true
		}
	case int32:
		if value > 0 {
			val = true
		}
	case int64:
		if value > 0 {
			val = true
		}
	case uint:
		if value > 0 {
			val = true
		}
	case uint8:
		if value > 0 {
			val = true
		}
	case uint16:
		if value > 0 {
			val = true
		}
	case uint32:
		if value > 0 {
			val = true
		}
	case uint64:
		if value > 0 {
			val = true
		}
	case bool:
		val = value
	case string:
		// for testing and other apps - numbers may appear as strings
		var err error
		if val, err = strconv.ParseBool(value); err != nil {
			return val
		}
	default:
		return false
	}
	return val
}
