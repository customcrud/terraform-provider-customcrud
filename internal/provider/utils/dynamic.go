package utils

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MapToDynamic converts a Go value to a types.Dynamic value.
func MapToDynamic(data interface{}) types.Dynamic {
	return types.DynamicValue(InterfaceToAttrValue(data))
}

// InterfaceToAttrValue converts a Go value to an attr.Value.
// InterfaceToAttrValue converts a Go value to an attr.Value.
func InterfaceToAttrValue(data interface{}) attr.Value {
	switch v := data.(type) {
	case string:
		return types.StringValue(v)
	case float64:
		return types.NumberValue(big.NewFloat(v))
	case int:
		return types.NumberValue(big.NewFloat(float64(v)))
	case bool:
		return types.BoolValue(v)
	case []interface{}:
		elements := make([]attr.Value, len(v))
		for i, elem := range v {
			elements[i] = InterfaceToAttrValue(elem)
		}
		tupleTypes := make([]attr.Type, len(elements))
		for i, elem := range elements {
			tupleTypes[i] = elem.Type(context.Background())
		}
		tupleVal, _ := types.TupleValue(tupleTypes, elements)
		return tupleVal
	case map[string]interface{}:
		attrs := make(map[string]attr.Value)
		attrTypes := make(map[string]attr.Type)
		for k, val := range v {
			attrs[k] = InterfaceToAttrValue(val)
			attrTypes[k] = attrs[k].Type(context.Background())
		}
		objVal, _ := types.ObjectValue(attrTypes, attrs)
		return objVal
	case nil:
		return types.DynamicNull()
	default:
		return types.StringValue(fmt.Sprintf("%v", v))
	}
}

// AttrValueToInterface converts an attr.Value to a Go value.
func AttrValueToInterface(val attr.Value) interface{} {
	switch v := val.(type) {
	case types.String:
		if v.IsNull() {
			return nil
		}
		return v.ValueString()
	case types.Number:
		if v.IsNull() {
			return nil
		}
		f, _ := v.ValueBigFloat().Float64()
		return f
	case types.Bool:
		if v.IsNull() {
			return nil
		}
		return v.ValueBool()
	case types.List:
		if v.IsNull() {
			return nil
		}
		elements := v.Elements()
		result := make([]interface{}, len(elements))
		for i, elem := range elements {
			if dynamicElem, ok := elem.(types.Dynamic); ok {
				result[i] = AttrValueToInterface(dynamicElem.UnderlyingValue())
			} else {
				result[i] = AttrValueToInterface(elem)
			}
		}
		return result
	case types.Tuple:
		if v.IsNull() {
			return nil
		}
		elements := v.Elements()
		result := make([]interface{}, len(elements))
		for i, elem := range elements {
			result[i] = AttrValueToInterface(elem)
		}
		return result
	case types.Object:
		if v.IsNull() {
			return nil
		}
		attrs := v.Attributes()
		result := make(map[string]interface{})
		for k, attr := range attrs {
			result[k] = AttrValueToInterface(attr)
		}
		return result
	case types.Dynamic:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return AttrValueToInterface(v.UnderlyingValue())
	default:
		return nil
	}
}
