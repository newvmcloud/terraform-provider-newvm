package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// PreventDecreaseInt64 errors if the planned value is less than the value in state.
// Useful for attributes like disk size where shrinking is not allowed.
type PreventDecreaseInt64 struct {
	// Optional: used only for nicer error messages.
	Attr string // e.g. "disk"
	Unit string // e.g. "GB"
}

func (m PreventDecreaseInt64) Description(_ context.Context) string {
	if m.Attr == "" {
		return "Prevents decreasing this value compared to the current state."
	}
	return fmt.Sprintf("Prevents decreasing %s compared to the current state.", m.Attr)
}

func (m PreventDecreaseInt64) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m PreventDecreaseInt64) PlanModifyInt64(
	_ context.Context,
	req planmodifier.Int64Request,
	resp *planmodifier.Int64Response,
) {
	// If either side is unknown/null, we can’t compare—do nothing.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() ||
		req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	oldV := req.StateValue.ValueInt64()
	newV := req.PlanValue.ValueInt64()

	if newV < oldV {
		attr := m.Attr
		if attr == "" {
			attr = "value"
		}
		unit := m.Unit
		if unit != "" {
			unit = " " + unit
		}

		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Decrease Not Allowed",
			fmt.Sprintf(
				"You attempted to reduce %s from %d%s to %d%s, which is not allowed.",
				attr, oldV, unit, newV, unit,
			),
		)
	}
}
