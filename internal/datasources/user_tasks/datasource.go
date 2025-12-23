package user_tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/inannamalick/terraform-provider-habitica/internal/client"
)

var (
	_ datasource.DataSource              = &userTasksDataSource{}
	_ datasource.DataSourceWithConfigure = &userTasksDataSource{}
)

// NewDataSource returns a new user_tasks data source.
func NewDataSource() datasource.DataSource {
	return &userTasksDataSource{}
}

type userTasksDataSource struct {
	client *client.Client
}

type userTasksModel struct {
	JSON types.String `tfsdk:"json"`
}

// Output types for JSON serialization
type tasksOutput struct {
	Dailies []dailyOutput `json:"dailies"`
	Habits  []habitOutput `json:"habits"`
	Todos   []todoOutput  `json:"todos"`
}

type dailyOutput struct {
	ID        string   `json:"id"`
	Text      string   `json:"text"`
	Notes     string   `json:"notes"`
	Completed bool     `json:"completed"`
	IsDue     bool     `json:"isDue"`
	Tags      []string `json:"tags"`
	Streak    int      `json:"streak"`
	Frequency string   `json:"frequency"`
}

type habitOutput struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Notes       string   `json:"notes"`
	CounterUp   int      `json:"counterUp"`
	CounterDown int      `json:"counterDown"`
	Tags        []string `json:"tags"`
}

type todoOutput struct {
	ID        string   `json:"id"`
	Text      string   `json:"text"`
	Notes     string   `json:"notes"`
	Completed bool     `json:"completed"`
	Tags      []string `json:"tags"`
}

func (d *userTasksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_tasks"
}

func (d *userTasksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all tasks (dailies, habits, todos) for the authenticated user with resolved tag names.",
		Attributes: map[string]schema.Attribute{
			"json": schema.StringAttribute{
				Description: "JSON output containing dailies, habits, and todos with resolved tag names.",
				Computed:    true,
			},
		},
	}
}

func (d *userTasksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *userTasksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Fetch all tasks
	tasks, err := d.client.GetAllTasks(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching tasks", err.Error())
		return
	}

	// Fetch all tags for UUID → name resolution
	tags, err := d.client.GetAllTags(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching tags", err.Error())
		return
	}

	// Build tag UUID → name map
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[tag.ID] = tag.Name
	}

	// Categorize and transform tasks
	output := tasksOutput{
		Dailies: []dailyOutput{},
		Habits:  []habitOutput{},
		Todos:   []todoOutput{},
	}

	for _, task := range tasks {
		// Resolve tag UUIDs to names
		resolvedTags := make([]string, 0, len(task.Tags))
		for _, tagID := range task.Tags {
			if name, ok := tagMap[tagID]; ok {
				resolvedTags = append(resolvedTags, name)
			}
		}

		switch task.Type {
		case "daily":
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
		case "habit":
			output.Habits = append(output.Habits, habitOutput{
				ID:          task.ID,
				Text:        task.Text,
				Notes:       task.Notes,
				CounterUp:   task.CounterUp,
				CounterDown: task.CounterDown,
				Tags:        resolvedTags,
			})
		case "todo":
			output.Todos = append(output.Todos, todoOutput{
				ID:        task.ID,
				Text:      task.Text,
				Notes:     task.Notes,
				Completed: task.Completed,
				Tags:      resolvedTags,
			})
		}
	}

	// Serialize to JSON
	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		resp.Diagnostics.AddError("Error serializing to JSON", err.Error())
		return
	}

	var state userTasksModel
	state.JSON = types.StringValue(string(jsonBytes))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
