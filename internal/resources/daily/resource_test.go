package daily

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/inannamalick/terraform-provider-habitica/internal/client"
	"github.com/stretchr/testify/assert"
)

// TestGetBoolWithDefault is a REGRESSION TEST for v0.2.1 bug
// where repeat field nested attributes with Computed+Default caused
// "Value Conversion Error" during import/refresh.
func TestGetBoolWithDefault(t *testing.T) {
	tests := []struct {
		name     string
		input    types.Bool
		defValue bool
		expected bool
	}{
		{
			name:     "Null returns default",
			input:    types.BoolNull(),
			defValue: true,
			expected: true,
		},
		{
			name:     "Unknown returns default",
			input:    types.BoolUnknown(),
			defValue: false,
			expected: false,
		},
		{
			name:     "True value overrides default",
			input:    types.BoolValue(true),
			defValue: false,
			expected: true,
		},
		{
			name:     "False value overrides default",
			input:    types.BoolValue(false),
			defValue: true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolWithDefault(tt.input, tt.defValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDailyRepeatConfigDefaults is a REGRESSION TEST for v0.2.1 bug
// Tests that when repeat block is not specified, Mon-Fri defaults are used
func TestDailyRepeatConfigDefaults(t *testing.T) {
	tests := []struct {
		name     string
		repeat   types.Object
		expected *client.RepeatConfig
	}{
		{
			name: "Nil repeat uses Mon-Fri defaults",
			repeat: types.ObjectNull(map[string]attr.Type{
				"monday":    types.BoolType,
				"tuesday":   types.BoolType,
				"wednesday": types.BoolType,
				"thursday":  types.BoolType,
				"friday":    types.BoolType,
				"saturday":  types.BoolType,
				"sunday":    types.BoolType,
			}),
			expected: &client.RepeatConfig{
				Monday:    true,
				Tuesday:   true,
				Wednesday: true,
				Thursday:  true,
				Friday:    true,
				Saturday:  false,
				Sunday:    false,
			},
		},
		{
			name: "Empty repeat uses Mon-Fri defaults",
			repeat: types.ObjectValueMust(
				map[string]attr.Type{
					"monday":    types.BoolType,
					"tuesday":   types.BoolType,
					"wednesday": types.BoolType,
					"thursday":  types.BoolType,
					"friday":    types.BoolType,
					"saturday":  types.BoolType,
					"sunday":    types.BoolType,
				},
				map[string]attr.Value{
					"monday":    types.BoolNull(),
					"tuesday":   types.BoolNull(),
					"wednesday": types.BoolNull(),
					"thursday":  types.BoolNull(),
					"friday":    types.BoolNull(),
					"saturday":  types.BoolNull(),
					"sunday":    types.BoolNull(),
				},
			),
			expected: &client.RepeatConfig{
				Monday:    true,
				Tuesday:   true,
				Wednesday: true,
				Thursday:  true,
				Friday:    true,
				Saturday:  false,
				Sunday:    false,
			},
		},
		{
			name: "Partial override Monday only",
			repeat: types.ObjectValueMust(
				map[string]attr.Type{
					"monday":    types.BoolType,
					"tuesday":   types.BoolType,
					"wednesday": types.BoolType,
					"thursday":  types.BoolType,
					"friday":    types.BoolType,
					"saturday":  types.BoolType,
					"sunday":    types.BoolType,
				},
				map[string]attr.Value{
					"monday":    types.BoolValue(false),
					"tuesday":   types.BoolNull(),
					"wednesday": types.BoolNull(),
					"thursday":  types.BoolNull(),
					"friday":    types.BoolNull(),
					"saturday":  types.BoolNull(),
					"sunday":    types.BoolNull(),
				},
			),
			expected: &client.RepeatConfig{
				Monday:    false, // overridden
				Tuesday:   true,
				Wednesday: true,
				Thursday:  true,
				Friday:    true,
				Saturday:  false,
				Sunday:    false,
			},
		},
		{
			name: "Enable weekend",
			repeat: types.ObjectValueMust(
				map[string]attr.Type{
					"monday":    types.BoolType,
					"tuesday":   types.BoolType,
					"wednesday": types.BoolType,
					"thursday":  types.BoolType,
					"friday":    types.BoolType,
					"saturday":  types.BoolType,
					"sunday":    types.BoolType,
				},
				map[string]attr.Value{
					"monday":    types.BoolNull(),
					"tuesday":   types.BoolNull(),
					"wednesday": types.BoolNull(),
					"thursday":  types.BoolNull(),
					"friday":    types.BoolNull(),
					"saturday":  types.BoolValue(true),
					"sunday":    types.BoolValue(true),
				},
			),
			expected: &client.RepeatConfig{
				Monday:    true,
				Tuesday:   true,
				Wednesday: true,
				Thursday:  true,
				Friday:    true,
				Saturday:  true, // enabled
				Sunday:    true, // enabled
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var repeatConfig *client.RepeatConfig

			if !tt.repeat.IsNull() && !tt.repeat.IsUnknown() {
				repeatAttrs := tt.repeat.Attributes()
				repeatConfig = &client.RepeatConfig{
					Monday:    getBoolFromObject(repeatAttrs, "monday", true),
					Tuesday:   getBoolFromObject(repeatAttrs, "tuesday", true),
					Wednesday: getBoolFromObject(repeatAttrs, "wednesday", true),
					Thursday:  getBoolFromObject(repeatAttrs, "thursday", true),
					Friday:    getBoolFromObject(repeatAttrs, "friday", true),
					Saturday:  getBoolFromObject(repeatAttrs, "saturday", false),
					Sunday:    getBoolFromObject(repeatAttrs, "sunday", false),
				}
			} else {
				// Default repeat config if not specified
				repeatConfig = &client.RepeatConfig{
					Monday:    true,
					Tuesday:   true,
					Wednesday: true,
					Thursday:  true,
					Friday:    true,
					Saturday:  false,
					Sunday:    false,
				}
			}

			assert.Equal(t, tt.expected, repeatConfig)
		})
	}
}

// TestDailyRepeatConfigAllExplicit tests that explicit values are respected
func TestDailyRepeatConfigAllExplicit(t *testing.T) {
	repeat := types.ObjectValueMust(
		map[string]attr.Type{
			"monday":    types.BoolType,
			"tuesday":   types.BoolType,
			"wednesday": types.BoolType,
			"thursday":  types.BoolType,
			"friday":    types.BoolType,
			"saturday":  types.BoolType,
			"sunday":    types.BoolType,
		},
		map[string]attr.Value{
			"monday":    types.BoolValue(false),
			"tuesday":   types.BoolValue(false),
			"wednesday": types.BoolValue(true),
			"thursday":  types.BoolValue(false),
			"friday":    types.BoolValue(false),
			"saturday":  types.BoolValue(true),
			"sunday":    types.BoolValue(true),
		},
	)

	repeatAttrs := repeat.Attributes()
	repeatConfig := &client.RepeatConfig{
		Monday:    getBoolFromObject(repeatAttrs, "monday", true),
		Tuesday:   getBoolFromObject(repeatAttrs, "tuesday", true),
		Wednesday: getBoolFromObject(repeatAttrs, "wednesday", true),
		Thursday:  getBoolFromObject(repeatAttrs, "thursday", true),
		Friday:    getBoolFromObject(repeatAttrs, "friday", true),
		Saturday:  getBoolFromObject(repeatAttrs, "saturday", false),
		Sunday:    getBoolFromObject(repeatAttrs, "sunday", false),
	}

	// Only Wednesday and weekend should be true
	assert.False(t, repeatConfig.Monday)
	assert.False(t, repeatConfig.Tuesday)
	assert.True(t, repeatConfig.Wednesday)
	assert.False(t, repeatConfig.Thursday)
	assert.False(t, repeatConfig.Friday)
	assert.True(t, repeatConfig.Saturday)
	assert.True(t, repeatConfig.Sunday)
}

// TestDailyModelToTaskConversion tests basic model conversion
func TestDailyModelToTaskConversion(t *testing.T) {
	model := &dailyResourceModel{
		Text:      types.StringValue("Morning Routine"),
		Notes:     types.StringValue("Brush teeth, shower"),
		Priority:  types.Float64Value(1.5),
		Frequency: types.StringValue("weekly"),
		EveryX:    types.Int64Value(1),
		Repeat: types.ObjectValueMust(
			map[string]attr.Type{
				"monday":    types.BoolType,
				"tuesday":   types.BoolType,
				"wednesday": types.BoolType,
				"thursday":  types.BoolType,
				"friday":    types.BoolType,
				"saturday":  types.BoolType,
				"sunday":    types.BoolType,
			},
			map[string]attr.Value{
				"monday":    types.BoolValue(true),
				"tuesday":   types.BoolValue(true),
				"wednesday": types.BoolValue(true),
				"thursday":  types.BoolValue(true),
				"friday":    types.BoolValue(true),
				"saturday":  types.BoolNull(), // Should default to false
				"sunday":    types.BoolNull(), // Should default to false
			},
		),
	}

	// Basic assertions on model
	assert.Equal(t, "Morning Routine", model.Text.ValueString())
	assert.Equal(t, "weekly", model.Frequency.ValueString())
	assert.False(t, model.Repeat.IsNull())

	// Test repeat defaults
	repeatAttrs := model.Repeat.Attributes()
	repeatConfig := &client.RepeatConfig{
		Monday:    getBoolFromObject(repeatAttrs, "monday", true),
		Tuesday:   getBoolFromObject(repeatAttrs, "tuesday", true),
		Wednesday: getBoolFromObject(repeatAttrs, "wednesday", true),
		Thursday:  getBoolFromObject(repeatAttrs, "thursday", true),
		Friday:    getBoolFromObject(repeatAttrs, "friday", true),
		Saturday:  getBoolFromObject(repeatAttrs, "saturday", false),
		Sunday:    getBoolFromObject(repeatAttrs, "sunday", false),
	}

	assert.True(t, repeatConfig.Monday)
	assert.True(t, repeatConfig.Friday)
	assert.False(t, repeatConfig.Saturday) // defaulted
	assert.False(t, repeatConfig.Sunday)   // defaulted
}
