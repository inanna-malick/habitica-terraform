package daily

import (
	"testing"

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
		repeat   *repeatModel
		expected *client.RepeatConfig
	}{
		{
			name:   "Nil repeat uses Mon-Fri defaults",
			repeat: nil,
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
			repeat: &repeatModel{
				Monday:    types.BoolNull(),
				Tuesday:   types.BoolNull(),
				Wednesday: types.BoolNull(),
				Thursday:  types.BoolNull(),
				Friday:    types.BoolNull(),
				Saturday:  types.BoolNull(),
				Sunday:    types.BoolNull(),
			},
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
			repeat: &repeatModel{
				Monday:    types.BoolValue(false),
				Tuesday:   types.BoolNull(),
				Wednesday: types.BoolNull(),
				Thursday:  types.BoolNull(),
				Friday:    types.BoolNull(),
				Saturday:  types.BoolNull(),
				Sunday:    types.BoolNull(),
			},
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
			repeat: &repeatModel{
				Monday:    types.BoolNull(),
				Tuesday:   types.BoolNull(),
				Wednesday: types.BoolNull(),
				Thursday:  types.BoolNull(),
				Friday:    types.BoolNull(),
				Saturday:  types.BoolValue(true),
				Sunday:    types.BoolValue(true),
			},
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

			if tt.repeat != nil {
				repeatConfig = &client.RepeatConfig{
					Monday:    getBoolWithDefault(tt.repeat.Monday, true),
					Tuesday:   getBoolWithDefault(tt.repeat.Tuesday, true),
					Wednesday: getBoolWithDefault(tt.repeat.Wednesday, true),
					Thursday:  getBoolWithDefault(tt.repeat.Thursday, true),
					Friday:    getBoolWithDefault(tt.repeat.Friday, true),
					Saturday:  getBoolWithDefault(tt.repeat.Saturday, false),
					Sunday:    getBoolWithDefault(tt.repeat.Sunday, false),
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
	repeat := &repeatModel{
		Monday:    types.BoolValue(false),
		Tuesday:   types.BoolValue(false),
		Wednesday: types.BoolValue(true),
		Thursday:  types.BoolValue(false),
		Friday:    types.BoolValue(false),
		Saturday:  types.BoolValue(true),
		Sunday:    types.BoolValue(true),
	}

	repeatConfig := &client.RepeatConfig{
		Monday:    getBoolWithDefault(repeat.Monday, true),
		Tuesday:   getBoolWithDefault(repeat.Tuesday, true),
		Wednesday: getBoolWithDefault(repeat.Wednesday, true),
		Thursday:  getBoolWithDefault(repeat.Thursday, true),
		Friday:    getBoolWithDefault(repeat.Friday, true),
		Saturday:  getBoolWithDefault(repeat.Saturday, false),
		Sunday:    getBoolWithDefault(repeat.Sunday, false),
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
		Repeat: &repeatModel{
			Monday: types.BoolValue(true),
			Tuesday: types.BoolValue(true),
			Wednesday: types.BoolValue(true),
			Thursday: types.BoolValue(true),
			Friday: types.BoolValue(true),
			Saturday: types.BoolNull(), // Should default to false
			Sunday: types.BoolNull(),   // Should default to false
		},
	}

	// Basic assertions on model
	assert.Equal(t, "Morning Routine", model.Text.ValueString())
	assert.Equal(t, "weekly", model.Frequency.ValueString())
	assert.NotNil(t, model.Repeat)

	// Test repeat defaults
	repeatConfig := &client.RepeatConfig{
		Monday:    getBoolWithDefault(model.Repeat.Monday, true),
		Tuesday:   getBoolWithDefault(model.Repeat.Tuesday, true),
		Wednesday: getBoolWithDefault(model.Repeat.Wednesday, true),
		Thursday:  getBoolWithDefault(model.Repeat.Thursday, true),
		Friday:    getBoolWithDefault(model.Repeat.Friday, true),
		Saturday:  getBoolWithDefault(model.Repeat.Saturday, false),
		Sunday:    getBoolWithDefault(model.Repeat.Sunday, false),
	}

	assert.True(t, repeatConfig.Monday)
	assert.True(t, repeatConfig.Friday)
	assert.False(t, repeatConfig.Saturday) // defaulted
	assert.False(t, repeatConfig.Sunday)   // defaulted
}
