package client

import "time"

// APIResponse is the standard Habitica API response envelope.
type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// Tag represents a Habitica tag.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Task represents a Habitica task (habit, daily, todo, or reward).
type Task struct {
	ID        string   `json:"id,omitempty"`
	Type      string   `json:"type"`
	Text      string   `json:"text"`
	Notes     string   `json:"notes,omitempty"`
	Alias     string   `json:"alias,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Priority  float64  `json:"priority,omitempty"`
	Attribute string   `json:"attribute,omitempty"`

	// Habit-specific fields
	Up          *bool `json:"up,omitempty"`
	Down        *bool `json:"down,omitempty"`
	CounterUp   int   `json:"counterUp,omitempty"`
	CounterDown int   `json:"counterDown,omitempty"`

	// Daily-specific fields
	Frequency    string        `json:"frequency,omitempty"`
	EveryX       int           `json:"everyX,omitempty"`
	StartDate    *time.Time    `json:"startDate,omitempty"`
	Repeat       *RepeatConfig `json:"repeat,omitempty"`
	DaysOfMonth  []int         `json:"daysOfMonth,omitempty"`
	WeeksOfMonth []int         `json:"weeksOfMonth,omitempty"`
	Streak       int           `json:"streak,omitempty"`
	IsDue        bool          `json:"isDue,omitempty"`
	NextDue      []string      `json:"nextDue,omitempty"`

	// Computed fields (read-only, gameplay-driven)
	Value float64 `json:"value,omitempty"`
}

// RepeatConfig defines which days of the week a daily repeats.
type RepeatConfig struct {
	Monday    bool `json:"m"`
	Tuesday   bool `json:"t"`
	Wednesday bool `json:"w"`
	Thursday  bool `json:"th"`
	Friday    bool `json:"f"`
	Saturday  bool `json:"s"`
	Sunday    bool `json:"su"`
}

// Webhook represents a Habitica webhook.
type Webhook struct {
	ID      string         `json:"id,omitempty"`
	URL     string         `json:"url"`
	Label   string         `json:"label,omitempty"`
	Type    string         `json:"type"`
	Enabled bool           `json:"enabled"`
	Options WebhookOptions `json:"options,omitempty"`
}

// WebhookOptions defines which events trigger the webhook.
type WebhookOptions struct {
	Created         bool `json:"created,omitempty"`
	Updated         bool `json:"updated,omitempty"`
	Deleted         bool `json:"deleted,omitempty"`
	Scored          bool `json:"scored,omitempty"`
	ChecklistScored bool `json:"checklistScored,omitempty"`
}
