package scripting

import (
	"github.com/d5/tengo/v2"
)

// goToTengo converts a Go value to a Tengo object.
func goToTengo(v interface{}) tengo.Object {
	switch val := v.(type) {
	case nil:
		return tengo.UndefinedValue
	case string:
		return &tengo.String{Value: val}
	case int:
		return &tengo.Int{Value: int64(val)}
	case int64:
		return &tengo.Int{Value: val}
	case float64:
		return &tengo.Float{Value: val}
	case bool:
		if val {
			return tengo.TrueValue
		}
		return tengo.FalseValue
	case map[string]interface{}:
		m := make(map[string]tengo.Object, len(val))
		for k, v := range val {
			m[k] = goToTengo(v)
		}
		return &tengo.ImmutableMap{Value: m}
	case map[string]string:
		m := make(map[string]tengo.Object, len(val))
		for k, v := range val {
			m[k] = &tengo.String{Value: v}
		}
		return &tengo.ImmutableMap{Value: m}
	case []interface{}:
		arr := make([]tengo.Object, len(val))
		for i, v := range val {
			arr[i] = goToTengo(v)
		}
		return &tengo.ImmutableArray{Value: arr}
	case []map[string]interface{}:
		arr := make([]tengo.Object, len(val))
		for i, v := range val {
			arr[i] = goToTengo(v)
		}
		return &tengo.ImmutableArray{Value: arr}
	default:
		obj, err := tengo.FromInterface(v)
		if err != nil {
			return tengo.UndefinedValue
		}
		return obj
	}
}

// tengoToGo converts a Tengo object to a Go value.
func tengoToGo(obj tengo.Object) interface{} {
	if obj == nil {
		return nil
	}
	switch val := obj.(type) {
	case *tengo.String:
		return val.Value
	case *tengo.Int:
		return val.Value
	case *tengo.Float:
		return val.Value
	case *tengo.Bool:
		return !val.IsFalsy()
	case *tengo.Map:
		return tengoMapToGo(val.Value)
	case *tengo.ImmutableMap:
		return tengoMapToGo(val.Value)
	case *tengo.Array:
		return tengoArrayToGo(val.Value)
	case *tengo.ImmutableArray:
		return tengoArrayToGo(val.Value)
	case *tengo.Undefined:
		return nil
	default:
		return obj.String()
	}
}

// tengoMapToGo converts a Tengo map to a Go map.
func tengoMapToGo(m map[string]tengo.Object) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = tengoToGo(v)
	}
	return result
}

// tengoArrayToGo converts a Tengo array to a Go slice.
func tengoArrayToGo(arr []tengo.Object) []interface{} {
	result := make([]interface{}, len(arr))
	for i, v := range arr {
		result[i] = tengoToGo(v)
	}
	return result
}

// tengoToStringMap extracts a map[string]string from a Tengo object.
func tengoToStringMap(obj tengo.Object) map[string]string {
	result := make(map[string]string)
	switch val := obj.(type) {
	case *tengo.Map:
		for k, v := range val.Value {
			if s, ok := v.(*tengo.String); ok {
				result[k] = s.Value
			}
		}
	case *tengo.ImmutableMap:
		for k, v := range val.Value {
			if s, ok := v.(*tengo.String); ok {
				result[k] = s.Value
			}
		}
	}
	return result
}

// normalizeForTengo recursively converts Go values to types that Tengo can handle.
// Specifically converts map[string]string and other typed maps to map[string]interface{}.
func normalizeForTengo(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, v := range val {
			result[k] = normalizeForTengo(v)
		}
		return result
	case map[string]string:
		result := make(map[string]interface{}, len(val))
		for k, v := range val {
			result[k] = v
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = normalizeForTengo(v)
		}
		return result
	case []map[string]interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = normalizeForTengo(v)
		}
		return result
	default:
		return v
	}
}

// getString extracts a string from a Tengo object, returning "" if not a string.
func getString(obj tengo.Object) string {
	if s, ok := obj.(*tengo.String); ok {
		return s.Value
	}
	return ""
}

// getInt extracts an int from a Tengo object, returning 0 if not numeric.
func getInt(obj tengo.Object) int {
	switch v := obj.(type) {
	case *tengo.Int:
		return int(v.Value)
	case *tengo.Float:
		return int(v.Value)
	default:
		return 0
	}
}

// getMap extracts the underlying map from a Tengo map object.
func getMap(obj tengo.Object) map[string]tengo.Object {
	switch v := obj.(type) {
	case *tengo.Map:
		return v.Value
	case *tengo.ImmutableMap:
		return v.Value
	default:
		return nil
	}
}
