package util

import (
	"strings"
)

type NestedData map[string]interface{}

func (d NestedData) Get(path string) (interface{}, bool) {
	keys := strings.Split(path, ".")
	current := d

	if len(keys) == 0 {
		return nil, false
	}

	val, exists := current[keys[0]]
	if !exists {
		return nil, false
	}

	if len(keys) == 1 {
		return val, true
	} else {
		next, ok := val.(map[string]interface{})
		if !ok {
			return nil, false
		}
		return NestedData(next).Get(strings.Join(keys[1:], "."))
	}
}

func (d NestedData) Set(path string, value interface{}) bool {
	keys := strings.Split(path, ".")
	current := d

	if len(keys) == 0 {
		return false
	}

	if len(keys) == 1 {
		current[keys[0]] = value
		return true
	} else {
		val, exists := current[keys[0]]
		if !exists {
			val = map[string]interface{}{}
			current[keys[0]] = val
		}

		next, ok := val.(map[string]interface{})
		if !ok {
			return false
		}

		return NestedData(next).Set(strings.Join(keys[1:], "."), value)
	}
}
