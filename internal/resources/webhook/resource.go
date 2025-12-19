package webhook

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/inannamalick/terraform-provider-habitica/internal/client"
)

var (
	_ resource.Resource                = &webhookResource{}
	_ resource.ResourceWithConfigure   = &webhookResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
)

// NewResource returns a new webhook resource.
func NewResource() resource.Resource {
	return &webhookResource{}
}

type webhookResource struct {
	client *client.Client
}

type webhookResourceModel struct {
	ID      types.String  `tfsdk:"id"`
	URL     types.String  `tfsdk:"url"`
	Label   types.String  `tfsdk:"label"`
	Type    types.String  `tfsdk:"type"`
	Enabled types.Bool    `tfsdk:"enabled"`
	Options *optionsModel `tfsdk:"options"`
}

type optionsModel struct {
	Created         types.Bool `tfsdk:"created"`
	Updated         types.Bool `tfsdk:"updated"`
	Deleted         types.Bool `tfsdk:"deleted"`
	Scored          types.Bool `tfsdk:"scored"`
	ChecklistScored types.Bool `tfsdk:"checklist_scored"`
}

func (r *webhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Habitica webhook for event notifications.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the webhook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Description: "The URL to send webhook notifications to.",
				Required:    true,
			},
			"label": schema.StringAttribute{
				Description: "A label for the webhook.",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of events to listen for: 'taskActivity', 'userActivity', 'questActivity', or 'groupChatReceived'.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the webhook is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"options": schema.SingleNestedAttribute{
				Description: "Event options for taskActivity webhooks.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"created": schema.BoolAttribute{
						Description: "Trigger on task creation.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"updated": schema.BoolAttribute{
						Description: "Trigger on task updates.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"deleted": schema.BoolAttribute{
						Description: "Trigger on task deletion.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"scored": schema.BoolAttribute{
						Description: "Trigger on task scoring.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"checklist_scored": schema.BoolAttribute{
						Description: "Trigger on checklist item scoring.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
				},
			},
		},
	}
}

func (r *webhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook := r.modelToWebhook(&plan)

	created, err := r.client.CreateWebhook(ctx, webhook)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	r.updateModelFromWebhook(&plan, created, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook, err := r.client.GetWebhook(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook", err.Error())
		return
	}

	r.updateModelFromWebhook(&state, webhook, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook := r.modelToWebhook(&plan)

	updated, err := r.client.UpdateWebhook(ctx, state.ID.ValueString(), webhook)
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook", err.Error())
		return
	}

	plan.ID = state.ID
	r.updateModelFromWebhook(&plan, updated, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWebhook(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting webhook", err.Error())
		return
	}
}

func (r *webhookResource) modelToWebhook(model *webhookResourceModel) *client.Webhook {
	webhook := &client.Webhook{
		URL:     model.URL.ValueString(),
		Label:   model.Label.ValueString(),
		Type:    model.Type.ValueString(),
		Enabled: model.Enabled.ValueBool(),
	}

	if model.Options != nil {
		webhook.Options = client.WebhookOptions{
			Created:         model.Options.Created.ValueBool(),
			Updated:         model.Options.Updated.ValueBool(),
			Deleted:         model.Options.Deleted.ValueBool(),
			Scored:          model.Options.Scored.ValueBool(),
			ChecklistScored: model.Options.ChecklistScored.ValueBool(),
		}
	}

	return webhook
}

func (r *webhookResource) updateModelFromWebhook(model *webhookResourceModel, webhook *client.Webhook, diags *diag.Diagnostics) {
	model.URL = types.StringValue(webhook.URL)
	model.Label = types.StringValue(webhook.Label)
	model.Type = types.StringValue(webhook.Type)
	model.Enabled = types.BoolValue(webhook.Enabled)

	model.Options = &optionsModel{
		Created:         types.BoolValue(webhook.Options.Created),
		Updated:         types.BoolValue(webhook.Options.Updated),
		Deleted:         types.BoolValue(webhook.Options.Deleted),
		Scored:          types.BoolValue(webhook.Options.Scored),
		ChecklistScored: types.BoolValue(webhook.Options.ChecklistScored),
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
