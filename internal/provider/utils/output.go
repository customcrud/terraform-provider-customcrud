package utils

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SplitSensitiveOutput moves keys listed in sensitive_outputs from output into output_sensitive.
func SplitSensitiveOutput(result map[string]interface{}, sensitiveOutputs types.List) (types.Dynamic, types.Dynamic) {
	regular := make(map[string]interface{}, len(result))
	for k, v := range result {
		regular[k] = v
	}
	sensitive := make(map[string]interface{})
	if !sensitiveOutputs.IsNull() && !sensitiveOutputs.IsUnknown() {
		for _, elem := range sensitiveOutputs.Elements() {
			k, ok := elem.(types.String)
			if !ok {
				continue
			}
			if v, ok := regular[k.ValueString()]; ok {
				sensitive[k.ValueString()] = v
				delete(regular, k.ValueString())
			}
		}
	}
	if len(sensitive) == 0 {
		return MapToDynamic(regular), types.DynamicNull()
	}
	return MapToDynamic(regular), MapToDynamic(sensitive)
}

// CombineOutput merges output and output_sensitive back into a single map for hook payloads.
// It returns nil (rather than an empty map) when neither is set, so the payload omits it.
func CombineOutput(output, outputSensitive types.Dynamic) interface{} {
	var combined map[string]interface{}
	for _, d := range []types.Dynamic{output, outputSensitive} {
		if d.IsNull() || d.IsUnknown() {
			continue
		}
		if m, ok := AttrValueToInterface(d.UnderlyingValue()).(map[string]interface{}); ok {
			if combined == nil {
				combined = make(map[string]interface{})
			}
			for k, v := range m {
				combined[k] = v
			}
		}
	}
	if combined == nil {
		return nil
	}
	return combined
}
