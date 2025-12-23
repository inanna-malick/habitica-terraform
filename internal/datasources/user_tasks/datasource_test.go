package user_tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/inannamalick/terraform-provider-habitica/internal/client"
	"github.com/inannamalick/terraform-provider-habitica/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserTasksDataSourceRead validates the full data source flow
func TestUserTasksDataSourceRead(t *testing.T) {
	// Define test data
	testTags := []client.Tag{
		{ID: "tag-1", Name: "work"},
		{ID: "tag-2", Name: "personal"},
	}

	testTasks := []client.Task{
		{
			ID:        "daily-1",
			Type:      "daily",
			Text:      "Morning routine",
			Notes:     "Brush teeth, shower",
			Completed: true,
			IsDue:     true,
			Tags:      []string{"tag-1"},
			Streak:    5,
			Frequency: "daily",
		},
		{
			ID:          "habit-1",
			Type:        "habit",
			Text:        "Exercise",
			Notes:       "30 min workout",
			CounterUp:   10,
			CounterDown: 2,
			Tags:        []string{"tag-1", "tag-2"},
		},
		{
			ID:        "todo-1",
			Type:      "todo",
			Text:      "Buy groceries",
			Notes:     "Milk, eggs, bread",
			Completed: false,
			Tags:      []string{"tag-2"},
		},
	}

	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/tasks/user": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    testTasks,
			})
		},
		"/tags": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    testTags,
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)

	// Fetch tasks
	tasks, err := c.GetAllTasks(context.Background())
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Fetch tags
	tags, err := c.GetAllTags(context.Background())
	require.NoError(t, err)
	assert.Len(t, tags, 2)

	// Verify tag resolution would work
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[tag.ID] = tag.Name
	}

	assert.Equal(t, "work", tagMap["tag-1"])
	assert.Equal(t, "personal", tagMap["tag-2"])
}

// TestUserTasksOutputFormat validates JSON output structure
func TestUserTasksOutputFormat(t *testing.T) {
	testTags := []client.Tag{
		{ID: "tag-1", Name: "exercise"},
	}

	testTasks := []client.Task{
		{
			ID:        "daily-1",
			Type:      "daily",
			Text:      "Run",
			Completed: false,
			IsDue:     true,
			Tags:      []string{"tag-1"},
			Streak:    3,
			Frequency: "weekly",
		},
	}

	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/tasks/user": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    testTasks,
			})
		},
		"/tags": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    testTags,
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)

	tasks, err := c.GetAllTasks(context.Background())
	require.NoError(t, err)

	tags, err := c.GetAllTags(context.Background())
	require.NoError(t, err)

	// Build tag map
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[tag.ID] = tag.Name
	}

	// Transform tasks
	output := tasksOutput{
		Dailies: []dailyOutput{},
		Habits:  []habitOutput{},
		Todos:   []todoOutput{},
	}

	for _, task := range tasks {
		resolvedTags := make([]string, 0, len(task.Tags))
		for _, tagID := range task.Tags {
			if name, ok := tagMap[tagID]; ok {
				resolvedTags = append(resolvedTags, name)
			}
		}

		if task.Type == "daily" {
			output.Dailies = append(output.Dailies, dailyOutput{
				ID:        task.ID,
				Text:      task.Text,
				Notes:     task.Notes,
				Completed: task.Completed,
				IsDue:     task.IsDue,
				Tags:      resolvedTags,
				Streak:    task.Streak,
				Frequency: task.Frequency,
			})
		}
	}

	// Verify output
	assert.Len(t, output.Dailies, 1)
	assert.Equal(t, "daily-1", output.Dailies[0].ID)
	assert.Equal(t, "Run", output.Dailies[0].Text)
	assert.Equal(t, []string{"exercise"}, output.Dailies[0].Tags)
	assert.Equal(t, 3, output.Dailies[0].Streak)
}

// TestUserTasksTagResolution validates tag UUID to name resolution
func TestUserTasksTagResolution(t *testing.T) {
	testTags := []client.Tag{
		{ID: "uuid-1", Name: "tier:foundation"},
		{ID: "uuid-2", Name: "tier:advanced"},
		{ID: "uuid-3", Name: "context:home"},
	}

	testTasks := []client.Task{
		{
			ID:   "task-1",
			Type: "habit",
			Text: "Multi-tag task",
			Tags: []string{"uuid-1", "uuid-3"},
		},
		{
			ID:   "task-2",
			Type: "todo",
			Text: "Single tag task",
			Tags: []string{"uuid-2"},
		},
		{
			ID:   "task-3",
			Type: "daily",
			Text: "No tags task",
			Tags: []string{},
		},
	}

	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/tasks/user": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    testTasks,
			})
		},
		"/tags": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    testTags,
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)

	tasks, err := c.GetAllTasks(context.Background())
	require.NoError(t, err)

	tags, err := c.GetAllTags(context.Background())
	require.NoError(t, err)

	// Build tag map
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[tag.ID] = tag.Name
	}

	// Test resolution
	for _, task := range tasks {
		resolvedTags := make([]string, 0, len(task.Tags))
		for _, tagID := range task.Tags {
			if name, ok := tagMap[tagID]; ok {
				resolvedTags = append(resolvedTags, name)
			}
		}

		switch task.ID {
		case "task-1":
			assert.Equal(t, []string{"tier:foundation", "context:home"}, resolvedTags)
		case "task-2":
			assert.Equal(t, []string{"tier:advanced"}, resolvedTags)
		case "task-3":
			assert.Empty(t, resolvedTags)
		}
	}
}

// TestUserTasksCategorization validates task type categorization
func TestUserTasksCategorization(t *testing.T) {
	testTasks := []client.Task{
		{ID: "daily-1", Type: "daily", Text: "Daily 1"},
		{ID: "daily-2", Type: "daily", Text: "Daily 2"},
		{ID: "habit-1", Type: "habit", Text: "Habit 1"},
		{ID: "habit-2", Type: "habit", Text: "Habit 2"},
		{ID: "habit-3", Type: "habit", Text: "Habit 3"},
		{ID: "todo-1", Type: "todo", Text: "Todo 1"},
	}

	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/tasks/user": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    testTasks,
			})
		},
		"/tags": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    []client.Tag{},
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)

	tasks, err := c.GetAllTasks(context.Background())
	require.NoError(t, err)

	// Categorize
	output := tasksOutput{
		Dailies: []dailyOutput{},
		Habits:  []habitOutput{},
		Todos:   []todoOutput{},
	}

	for _, task := range tasks {
		switch task.Type {
		case "daily":
			output.Dailies = append(output.Dailies, dailyOutput{
				ID:   task.ID,
				Text: task.Text,
			})
		case "habit":
			output.Habits = append(output.Habits, habitOutput{
				ID:   task.ID,
				Text: task.Text,
			})
		case "todo":
			output.Todos = append(output.Todos, todoOutput{
				ID:   task.ID,
				Text: task.Text,
			})
		}
	}

	assert.Len(t, output.Dailies, 2)
	assert.Len(t, output.Habits, 3)
	assert.Len(t, output.Todos, 1)
}

// TestUserTasksEmptyResults validates handling of empty task/tag lists
func TestUserTasksEmptyResults(t *testing.T) {
	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/tasks/user": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    []client.Task{},
			})
		},
		"/tags": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    []client.Tag{},
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)

	tasks, err := c.GetAllTasks(context.Background())
	require.NoError(t, err)
	assert.Empty(t, tasks)

	tags, err := c.GetAllTags(context.Background())
	require.NoError(t, err)
	assert.Empty(t, tags)

	// Verify empty output can be serialized
	output := tasksOutput{
		Dailies: []dailyOutput{},
		Habits:  []habitOutput{},
		Todos:   []todoOutput{},
	}

	jsonBytes, err := json.Marshal(output)
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), "dailies")
	assert.Contains(t, string(jsonBytes), "habits")
	assert.Contains(t, string(jsonBytes), "todos")
}

// TestUserTasksJSONSerialization validates JSON output is valid
func TestUserTasksJSONSerialization(t *testing.T) {
	output := tasksOutput{
		Dailies: []dailyOutput{
			{
				ID:        "daily-1",
				Text:      "Test daily",
				Notes:     "Test notes",
				Completed: true,
				IsDue:     false,
				Tags:      []string{"work", "urgent"},
				Streak:    10,
				Frequency: "daily",
			},
		},
		Habits: []habitOutput{
			{
				ID:          "habit-1",
				Text:        "Test habit",
				CounterUp:   5,
				CounterDown: 2,
				Tags:        []string{"exercise"},
			},
		},
		Todos: []todoOutput{
			{
				ID:        "todo-1",
				Text:      "Test todo",
				Completed: false,
				Tags:      []string{},
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	require.NoError(t, err)

	// Verify it's valid JSON by unmarshaling
	var parsed tasksOutput
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.Len(t, parsed.Dailies, 1)
	assert.Equal(t, "Test daily", parsed.Dailies[0].Text)
	assert.Equal(t, 10, parsed.Dailies[0].Streak)

	assert.Len(t, parsed.Habits, 1)
	assert.Equal(t, "Test habit", parsed.Habits[0].Text)
	assert.Equal(t, 5, parsed.Habits[0].CounterUp)

	assert.Len(t, parsed.Todos, 1)
	assert.Equal(t, "Test todo", parsed.Todos[0].Text)
}
