// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// useStateForUnknownIfInputUnchanged uses the prior state value for the output
// attribute when the input attribute hasn't changed. This prevents unnecessary
// "(known after apply)" in plans when no CRUD hook will execute.
type useStateForUnknownIfInputUnchanged struct{}

func UseStateForUnknownIfInputUnchanged() planmodifier.Dynamic {
	return useStateForUnknownIfInputUnchanged{}
}

func (m useStateForUnknownIfInputUnchanged) Description(_ context.Context) string {
	return "Uses the prior state value when input hasn't changed, as the output won't change without a CRUD operation."
}

func (m useStateForUnknownIfInputUnchanged) MarkdownDescription(_ context.Context) string {
	return "Uses the prior state value when input hasn't changed, as the output won't change without a CRUD operation."
}

func (m useStateForUnknownIfInputUnchanged) PlanModifyDynamic(ctx context.Context, req planmodifier.DynamicRequest, resp *planmodifier.DynamicResponse) {
	// Do nothing during create (no prior state).
	if req.State.Raw.IsNull() {
		return
	}

	// Do nothing during destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	// Do nothing if output is already known.
	if !req.PlanValue.IsUnknown() && !req.PlanValue.IsUnderlyingValueUnknown() {
		return
	}

	// Read input from state and plan to compare.
	var stateInput, planInput types.Dynamic
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("input"), &stateInput)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("input"), &planInput)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If input changed, leave output as unknown (hook will produce new values).
	if !stateInput.Equal(planInput) {
		return
	}

	// Input unchanged — use prior state value for output.
	resp.PlanValue = req.StateValue
}
