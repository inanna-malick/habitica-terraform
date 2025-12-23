package daily

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/inannamalick/terraform-provider-habitica/internal/client"
)

var (
	_ resource.Resource                = &dailyResource{}
	_ resource.ResourceWithConfigure   = &dailyResource{}
	_ resource.ResourceWithImportState = &dailyResource{}
)

// NewResource returns a new daily resource.
func NewResource() resource.Resource {
	return &dailyResource{}
}

type dailyResource struct {
	client *client.Client
}

type dailyResourceModel struct {
	ID           types.String  `tfsdk:"id"`
	Text         types.String  `tfsdk:"text"`
	Notes        types.String  `tfsdk:"notes"`
	Priority     types.Float64 `tfsdk:"priority"`
	Frequency    types.String  `tfsdk:"frequency"`
	EveryX       types.Int64   `tfsdk:"every_x"`
	StartDate    types.String  `tfsdk:"start_date"`
	Repeat       *repeatModel  `tfsdk:"repeat"`
	DaysOfMonth  types.List    `tfsdk:"days_of_month"`
	WeeksOfMonth types.List    `tfsdk:"weeks_of_month"`
	Tags         types.List    `tfsdk:"tags"`
}

type repeatModel struct {
	Monday    types.Bool `tfsdk:"monday"`
	Tuesday   types.Bool `tfsdk:"tuesday"`
	Wednesday types.Bool `tfsdk:"wednesday"`
	Thursday  types.Bool `tfsdk:"thursday"`
	Friday    types.Bool `tfsdk:"friday"`
	Saturday  types.Bool `tfsdk:"saturday"`
	Sunday    types.Bool `tfsdk:"sunday"`
}

func (r *dailyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_daily"
}

func (r *dailyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Habitica daily (recurring task).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the daily.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"text": schema.StringAttribute{
				Description: "The title of the daily.",
				Required:    true,
			},
			"notes": schema.StringAttribute{
				Description: "Extra notes or description for the daily.",
				Optional:    true,
				Computed:    true,
			},
			"priority": schema.Float64Attribute{
				Description: "Difficulty level: 0.1 (trivial), 1 (easy), 1.5 (medium), 2 (hard). Defaults to 1.",
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(1),
			},
			"frequency": schema.StringAttribute{
				Description: "Repeat frequency: 'daily', 'weekly', 'monthly', or 'yearly'. Defaults to 'weekly'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("weekly"),
			},
			"every_x": schema.Int64Attribute{
				Description: "Repeat every X periods (e.g., every 2 weeks). Defaults to 1.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
			},
			"start_date": schema.StringAttribute{
				Description: "Start date in YYYY-MM-DD format. Defaults to today.",
				Optional:    true,
				Computed:    true,
			},
			"repeat": schema.SingleNestedAttribute{
				Description: "Which days of the week the daily repeats (for weekly frequency).",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"monday": schema.BoolAttribute{
						Description: "Repeat on Monday. Defaults to true if not specified.",
						Optional:    true,
					},
					"tuesday": schema.BoolAttribute{
						Description: "Repeat on Tuesday. Defaults to true if not specified.",
						Optional:    true,
					},
					"wednesday": schema.BoolAttribute{
						Description: "Repeat on Wednesday. Defaults to true if not specified.",
						Optional:    true,
					},
					"thursday": schema.BoolAttribute{
						Description: "Repeat on Thursday. Defaults to true if not specified.",
						Optional:    true,
					},
					"friday": schema.BoolAttribute{
						Description: "Repeat on Friday. Defaults to true if not specified.",
						Optional:    true,
					},
					"saturday": schema.BoolAttribute{
						Description: "Repeat on Saturday. Defaults to false if not specified.",
						Optional:    true,
					},
					"sunday": schema.BoolAttribute{
						Description: "Repeat on Sunday. Defaults to false if not specified.",
						Optional:    true,
					},
				},
			},
			"days_of_month": schema.ListAttribute{
				Description: "Days of the month to repeat on (for monthly frequency).",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"weeks_of_month": schema.ListAttribute{
				Description: "Weeks of the month to repeat on (1-5, for monthly frequency).",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"tags": schema.ListAttribute{
				Description: "List of tag IDs to associate with this daily.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *dailyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *dailyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dailyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	task := r.modelToTask(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateTask(ctx, task)
	if err != nil {
		resp.Diagnostics.AddError("Error creating daily", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	r.updateModelFromTask(ctx, &plan, created, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *dailyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dailyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	task, err := r.client.GetTask(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading daily", err.Error())
		return
	}

	r.updateModelFromTask(ctx, &state, task, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *dailyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dailyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state dailyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	task := r.modelToTask(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateTask(ctx, state.ID.ValueString(), task)
	if err != nil {
		resp.Diagnostics.AddError("Error updating daily", err.Error())
		return
	}

	plan.ID = state.ID
	r.updateModelFromTask(ctx, &plan, updated, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *dailyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dailyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTask(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting daily", err.Error())
		return
	}
}

func (r *dailyResource) modelToTask(ctx context.Context, model *dailyResourceModel, diags *diag.Diagnostics) *client.Task {
	task := &client.Task{
		Type:      "daily",
		Text:      model.Text.ValueString(),
		Notes:     model.Notes.ValueString(),
		Priority:  model.Priority.ValueFloat64(),
		Frequency: model.Frequency.ValueString(),
		EveryX:    int(model.EveryX.ValueInt64()),
	}

	if !model.StartDate.IsNull() && !model.StartDate.IsUnknown() {
		t, err := time.Parse("2006-01-02", model.StartDate.ValueString())
		if err == nil {
			task.StartDate = &t
		}
	}

	// Handle repeat config with defaults
	if model.Repeat != nil {
		task.Repeat = &client.RepeatConfig{
			Monday:    getBoolWithDefault(model.Repeat.Monday, true),
			Tuesday:   getBoolWithDefault(model.Repeat.Tuesday, true),
			Wednesday: getBoolWithDefault(model.Repeat.Wednesday, true),
			Thursday:  getBoolWithDefault(model.Repeat.Thursday, true),
			Friday:    getBoolWithDefault(model.Repeat.Friday, true),
			Saturday:  getBoolWithDefault(model.Repeat.Saturday, false),
			Sunday:    getBoolWithDefault(model.Repeat.Sunday, false),
		}
	} else {
		// Default repeat config if not specified: Mon-Fri
		task.Repeat = &client.RepeatConfig{
			Monday:    true,
			Tuesday:   true,
			Wednesday: true,
			Thursday:  true,
			Friday:    true,
			Saturday:  false,
			Sunday:    false,
		}
	}

	if !model.DaysOfMonth.IsNull() {
		var days []int64
		diags.Append(model.DaysOfMonth.ElementsAs(ctx, &days, false)...)
		for _, d := range days {
			task.DaysOfMonth = append(task.DaysOfMonth, int(d))
		}
	}

	if !model.WeeksOfMonth.IsNull() {
		var weeks []int64
		diags.Append(model.WeeksOfMonth.ElementsAs(ctx, &weeks, false)...)
		for _, w := range weeks {
			task.WeeksOfMonth = append(task.WeeksOfMonth, int(w))
		}
	}

	if !model.Tags.IsNull() {
		var tags []string
		diags.Append(model.Tags.ElementsAs(ctx, &tags, false)...)
		task.Tags = tags
	}

	return task
}

func (r *dailyResource) updateModelFromTask(ctx context.Context, model *dailyResourceModel, task *client.Task, diags *diag.Diagnostics) {
	model.Text = types.StringValue(task.Text)
	model.Notes = types.StringValue(task.Notes)
	model.Priority = types.Float64Value(task.Priority)
	model.Frequency = types.StringValue(task.Frequency)
	model.EveryX = types.Int64Value(int64(task.EveryX))

	if task.StartDate != nil {
		model.StartDate = types.StringValue(task.StartDate.Format("2006-01-02"))
	}

	if task.Repeat != nil {
		model.Repeat = &repeatModel{
			Monday:    types.BoolValue(task.Repeat.Monday),
			Tuesday:   types.BoolValue(task.Repeat.Tuesday),
			Wednesday: types.BoolValue(task.Repeat.Wednesday),
			Thursday:  types.BoolValue(task.Repeat.Thursday),
			Friday:    types.BoolValue(task.Repeat.Friday),
			Saturday:  types.BoolValue(task.Repeat.Saturday),
			Sunday:    types.BoolValue(task.Repeat.Sunday),
		}
	}

	if len(task.DaysOfMonth) > 0 {
		days := make([]int64, len(task.DaysOfMonth))
		for i, d := range task.DaysOfMonth {
			days[i] = int64(d)
		}
		daysList, d := types.ListValueFrom(ctx, types.Int64Type, days)
		diags.Append(d...)
		model.DaysOfMonth = daysList
	} else {
		model.DaysOfMonth = types.ListNull(types.Int64Type)
	}

	if len(task.WeeksOfMonth) > 0 {
		weeks := make([]int64, len(task.WeeksOfMonth))
		for i, w := range task.WeeksOfMonth {
			weeks[i] = int64(w)
		}
		weeksList, d := types.ListValueFrom(ctx, types.Int64Type, weeks)
		diags.Append(d...)
		model.WeeksOfMonth = weeksList
	} else {
		model.WeeksOfMonth = types.ListNull(types.Int64Type)
	}

	if len(task.Tags) > 0 {
		tagList, d := types.ListValueFrom(ctx, types.StringType, task.Tags)
		diags.Append(d...)
		model.Tags = tagList
	} else {
		model.Tags = types.ListNull(types.StringType)
	}
}

func (r *dailyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// getBoolWithDefault returns the bool value if not null, otherwise returns the default
func getBoolWithDefault(val types.Bool, defaultVal bool) bool {
	if val.IsNull() || val.IsUnknown() {
		return defaultVal
	}
	return val.ValueBool()
}
