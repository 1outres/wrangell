package v1alpha1

import (
	"fmt"
	"github.com/1outres/wrangell/internal/util"
	"strings"
)

type DataSchema struct {
	Fields []DataSchemaField `json:"fields"`
}

type DataSchemaField struct {
	Name string `json:"name"`
	//+kubebuilder:validation:Enum=string;int;float;bool;string_list;int_list;float_list
	Type FieldType `json:"type"`
}

type FieldType string

const (
	FieldTypeString     FieldType = "string"
	FieldTypeInt        FieldType = "int"
	FieldTypeFloat      FieldType = "float"
	FieldTypeBool       FieldType = "bool"
	FieldTypeStringList FieldType = "string_list"
	FieldTypeIntList    FieldType = "int_list"
	FieldTypeFloatList  FieldType = "float_list"
)

func (ds *DataSchema) Validate() error {
	structure := map[string]string{}

	for _, field := range ds.Fields {
		keys := strings.Split(field.Name, ".")

		for i, _ := range keys {
			if i == len(keys)-1 {
				if _, exists := structure[field.Name]; exists {
					return fmt.Errorf("field %s is duplicated", field.Name)
				} else {
					structure[field.Name] = string(field.Type)
				}
			} else {
				path := strings.Join(keys[:i+1], ".")
				if _, exists := structure[path]; !exists {
					structure[path] = "object"
				} else {
					if structure[path] != "object" {
						return fmt.Errorf("field %s is not an object", path)
					}
				}
			}
		}
	}

	return nil
}

func (ds *DataSchema) CreateEmptyData() util.NestedData {
	data := util.NestedData{}

	for _, field := range ds.Fields {
		data.Set(field.Name, field.GetDefaultValue())
	}

	return data
}

func (ds *DataSchema) CreateDataFrom(source util.NestedData) (util.NestedData, error) {
	data := ds.CreateEmptyData()

	for _, field := range ds.Fields {
		value, exists := source.Get(field.Name)
		if !exists {
			return nil, fmt.Errorf("field %s is missing", field.Name)
		}

		if success := data.Set(field.Name, value); !success {
			return nil, fmt.Errorf("failed to set field %s", field.Name)
		}
	}

	return data, nil
}

func (ds *DataSchema) GetField(name string) (*DataSchemaField, bool) {
	for _, field := range ds.Fields {
		if field.Name == name {
			return &field, true
		}
	}

	return nil, false
}

func (df *DataSchemaField) GetDefaultValue() interface{} {
	switch df.Type {
	case FieldTypeString:
		return ""
	case FieldTypeInt:
		return 0
	case FieldTypeFloat:
		return 0.0
	case FieldTypeBool:
		return false
	case FieldTypeStringList:
		return []string{}
	case FieldTypeIntList:
		return []int{}
	case FieldTypeFloatList:
		return []float64{}
	default:
		return nil
	}
}
