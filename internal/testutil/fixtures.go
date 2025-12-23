package testutil

import (
	"time"

	"github.com/inannamalick/terraform-provider-habitica/internal/client"
)

// Test fixtures for common test data

var (
	// Tags
	TestTag1 = client.Tag{
		ID:   "tag-uuid-1",
		Name: "work",
	}

	TestTag2 = client.Tag{
		ID:   "tag-uuid-2",
		Name: "exercise",
	}

	TestTag3 = client.Tag{
		ID:   "tag-uuid-3",
		Name: "tier:foundation",
	}

	// Habits
	TestHabit1 = client.Task{
		ID:       "habit-uuid-1",
		Type:     "habit",
		Text:     "Exercise",
		Notes:    "Get moving",
		Priority: 1.5,
		Up:       BoolPtr(true),
		Down:     BoolPtr(false),
		Tags:     []string{"tag-uuid-2"},
	}

	TestHabit2 = client.Task{
		ID:       "habit-uuid-2",
		Type:     "habit",
		Text:     "Drink water",
		Priority: 1.0,
		Up:       BoolPtr(true),
		Down:     BoolPtr(true),
		Tags:     []string{},
	}

	// Dailies
	TestDaily1 = client.Task{
		ID:        "daily-uuid-1",
		Type:      "daily",
		Text:      "Morning routine",
		Notes:     "Brush teeth, shower, etc.",
		Priority:  1.5,
		Frequency: "weekly",
		EveryX:    1,
		StartDate: TimePtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		Repeat: &client.RepeatConfig{
			Monday:    true,
			Tuesday:   true,
			Wednesday: true,
			Thursday:  true,
			Friday:    true,
			Saturday:  false,
			Sunday:    false,
		},
		Tags:      []string{"tag-uuid-3"},
		Completed: false,
		IsDue:     true,
		Streak:    5,
	}

	TestDaily2 = client.Task{
		ID:        "daily-uuid-2",
		Type:      "daily",
		Text:      "Take meds",
		Frequency: "daily",
		EveryX:    1,
		Repeat: &client.RepeatConfig{
			Monday:    true,
			Tuesday:   true,
			Wednesday: true,
			Thursday:  true,
			Friday:    true,
			Saturday:  true,
			Sunday:    true,
		},
		Tags:      []string{},
		Completed: true,
		IsDue:     false,
		Streak:    30,
	}

	// Webhooks
	TestWebhook1 = client.Webhook{
		ID:      "webhook-uuid-1",
		URL:     "https://example.com/hook",
		Label:   "Test Webhook",
		Type:    "taskActivity",
		Enabled: true,
		Options: client.WebhookOptions{
			Created: true,
			Updated: true,
			Deleted: false,
			Scored:  true,
		},
	}

	TestWebhook2 = client.Webhook{
		ID:      "webhook-uuid-2",
		URL:     "https://example.com/hook2",
		Label:   "All Events",
		Type:    "taskActivity",
		Enabled: false,
		Options: client.WebhookOptions{
			Created:         true,
			Updated:         true,
			Deleted:         true,
			Scored:          true,
			ChecklistScored: true,
		},
	}
)
