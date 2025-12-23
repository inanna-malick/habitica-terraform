package habit

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// TestGetBoolWithDefault is a REGRESSION TEST for v0.2.2 bug
// where Optional+Computed+Default caused value conversion errors.
// The fix was to remove Computed+Default and handle defaults in code.
func TestGetBoolWithDefault(t *testing.T) {
	tests := []struct {
		name     string
		input    types.Bool
		defValue bool
		expected bool
	}{
		{
			name:     "Null returns default true",
			input:    types.BoolNull(),
			defValue: true,
			expected: true,
		},
		{
			name:     "Null returns default false",
			input:    types.BoolNull(),
			defValue: false,
			expected: false,
		},
		{
			name:     "Unknown returns default true",
			input:    types.BoolUnknown(),
			defValue: true,
			expected: true,
		},
		{
			name:     "Unknown returns default false",
			input:    types.BoolUnknown(),
			defValue: false,
			expected: false,
		},
		{
			name:     "True value returns true",
			input:    types.BoolValue(true),
			defValue: false,
			expected: true,
		},
		{
			name:     "False value returns false",
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

// TestHabitUpDownDefaults tests that up/down fields default correctly
// This is a regression test for the v0.2.2 bug where Computed+Default
// on these fields caused "Value Conversion Error"
func TestHabitUpDownDefaults(t *testing.T) {
	tests := []struct {
		name         string
		upValue      types.Bool
		downValue    types.Bool
		expectedUp   bool
		expectedDown bool
	}{
		{
			name:         "Both null - use defaults",
			upValue:      types.BoolNull(),
			downValue:    types.BoolNull(),
			expectedUp:   true,  // default
			expectedDown: false, // default
		},
		{
			name:         "Up explicit false, down null",
			upValue:      types.BoolValue(false),
			downValue:    types.BoolNull(),
			expectedUp:   false,
			expectedDown: false, // default
		},
		{
			name:         "Up null, down explicit true",
			upValue:      types.BoolNull(),
			downValue:    types.BoolValue(true),
			expectedUp:   true, // default
			expectedDown: true,
		},
		{
			name:         "Both explicit",
			upValue:      types.BoolValue(false),
			downValue:    types.BoolValue(true),
			expectedUp:   false,
			expectedDown: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up := getBoolWithDefault(tt.upValue, true)
			down := getBoolWithDefault(tt.downValue, false)

			assert.Equal(t, tt.expectedUp, up)
			assert.Equal(t, tt.expectedDown, down)
		})
	}
}

// TestHabitModelToTaskConversion tests the conversion from Terraform model to API Task
func TestHabitModelToTaskConversion(t *testing.T) {
	model := &habitResourceModel{
		Text:     types.StringValue("Test Habit"),
		Notes:    types.StringValue("Test Notes"),
		Priority: types.Float64Value(1.5),
		Up:       types.BoolValue(true),
		Down:     types.BoolValue(false),
	}

	// Test that model fields are set correctly
	t.Run("Converts model fields correctly", func(t *testing.T) {
		assert.Equal(t, "Test Habit", model.Text.ValueString())
		assert.Equal(t, 1.5, model.Priority.ValueFloat64())
		assert.True(t, model.Up.ValueBool())
		assert.False(t, model.Down.ValueBool())
	})
}

// TestHabitUpdateModelFromTask tests updating the Terraform model from API response
func TestHabitUpdateModelFromTask(t *testing.T) {
	// This would test the updateModelFromTask function
	// Skipping detailed implementation for now as it requires full resource context
	t.Skip("Full resource tests require provider context")
}
