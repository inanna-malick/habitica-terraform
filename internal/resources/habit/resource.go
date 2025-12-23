package habit

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/inannamalick/terraform-provider-habitica/internal/client"
)

var (
	_ resource.Resource                = &habitResource{}
	_ resource.ResourceWithConfigure   = &habitResource{}
	_ resource.ResourceWithImportState = &habitResource{}
)

// NewResource returns a new habit resource.
func NewResource() resource.Resource {
	return &habitResource{}
}

type habitResource struct {
	client *client.Client
}

type habitResourceModel struct {
	ID       types.String  `tfsdk:"id"`
	Text     types.String  `tfsdk:"text"`
	Notes    types.String  `tfsdk:"notes"`
	Priority types.Float64 `tfsdk:"priority"`
	Up       types.Bool    `tfsdk:"up"`
	Down     types.Bool    `tfsdk:"down"`
	Tags     types.List    `tfsdk:"tags"`
}

func (r *habitResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_habit"
}

func (r *habitResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Habitica habit.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the habit.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"text": schema.StringAttribute{
				Description: "The title of the habit.",
				Required:    true,
			},
			"notes": schema.StringAttribute{
				Description: "Extra notes or description for the habit.",
				Optional:    true,
				Computed:    true,
			},
			"priority": schema.Float64Attribute{
				Description: "Difficulty level: 0.1 (trivial), 1 (easy), 1.5 (medium), 2 (hard). Defaults to 1.",
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(1),
			},
			"up": schema.BoolAttribute{
				Description: "Whether the habit can be scored positively (+). Defaults to true if not specified.",
				Optional:    true,
				Computed:    true,
			},
			"down": schema.BoolAttribute{
				Description: "Whether the habit can be scored negatively (-). Defaults to false if not specified.",
				Optional:    true,
				Computed:    true,
			},
			"tags": schema.ListAttribute{
				Description: "List of tag IDs to associate with this habit.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *habitResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *habitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan habitResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle defaults for up/down
	up := getBoolWithDefault(plan.Up, true)
	down := getBoolWithDefault(plan.Down, false)

	task := &client.Task{
		Type:     "habit",
		Text:     plan.Text.ValueString(),
		Notes:    plan.Notes.ValueString(),
		Priority: plan.Priority.ValueFloat64(),
		Up:       &up,
		Down:     &down,
	}

	if !plan.Tags.IsNull() {
		var tags []string
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		task.Tags = tags
	}

	created, err := r.client.CreateTask(ctx, task)
	if err != nil {
		resp.Diagnostics.AddError("Error creating habit", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	r.updateModelFromTask(ctx, &plan, created, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *habitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state habitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	task, err := r.client.GetTask(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading habit", err.Error())
		return
	}

	r.updateModelFromTask(ctx, &state, task, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *habitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan habitResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state habitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle defaults for up/down
	up := getBoolWithDefault(plan.Up, true)
	down := getBoolWithDefault(plan.Down, false)

	task := &client.Task{
		Text:     plan.Text.ValueString(),
		Notes:    plan.Notes.ValueString(),
		Priority: plan.Priority.ValueFloat64(),
		Up:       &up,
		Down:     &down,
	}

	if !plan.Tags.IsNull() {
		var tags []string
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		task.Tags = tags
	} else {
		task.Tags = []string{}
	}

	updated, err := r.client.UpdateTask(ctx, state.ID.ValueString(), task)
	if err != nil {
		resp.Diagnostics.AddError("Error updating habit", err.Error())
		return
	}

	plan.ID = state.ID
	r.updateModelFromTask(ctx, &plan, updated, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *habitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state habitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTask(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting habit", err.Error())
		return
	}
}

func (r *habitResource) updateModelFromTask(ctx context.Context, model *habitResourceModel, task *client.Task, diags *diag.Diagnostics) {
	model.Text = types.StringValue(task.Text)
	model.Notes = types.StringValue(task.Notes)
	model.Priority = types.Float64Value(task.Priority)

	if task.Up != nil {
		model.Up = types.BoolValue(*task.Up)
	}
	if task.Down != nil {
		model.Down = types.BoolValue(*task.Down)
	}

	if len(task.Tags) > 0 {
		tagList, d := types.ListValueFrom(ctx, types.StringType, task.Tags)
		diags.Append(d...)
		model.Tags = tagList
	} else {
		model.Tags = types.ListNull(types.StringType)
	}
}

func (r *habitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// getBoolWithDefault returns the bool value if not null, otherwise returns the default
func getBoolWithDefault(val types.Bool, defaultVal bool) bool {
	if val.IsNull() || val.IsUnknown() {
		return defaultVal
	}
	return val.ValueBool()
}
