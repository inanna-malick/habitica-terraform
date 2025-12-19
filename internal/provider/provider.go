package provider

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/inannamalick/terraform-provider-habitica/internal/client"
	"github.com/inannamalick/terraform-provider-habitica/internal/resources/daily"
	"github.com/inannamalick/terraform-provider-habitica/internal/resources/habit"
	"github.com/inannamalick/terraform-provider-habitica/internal/resources/tag"
	"github.com/inannamalick/terraform-provider-habitica/internal/resources/webhook"
)

var _ provider.Provider = &HabiticaProvider{}

// HabiticaProvider defines the provider implementation.
type HabiticaProvider struct {
	version string
}

// HabiticaProviderModel describes the provider data model.
type HabiticaProviderModel struct {
	UserID          types.String `tfsdk:"user_id"`
	APIToken        types.String `tfsdk:"api_token"`
	ClientAuthorID  types.String `tfsdk:"client_author_id"`
	ClientAppName   types.String `tfsdk:"client_app_name"`
	RateLimitBuffer types.Int64  `tfsdk:"rate_limit_buffer"`
}

// New returns a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HabiticaProvider{
			version: version,
		}
	}
}

func (p *HabiticaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "habitica"
	resp.Version = p.version
}

func (p *HabiticaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Habitica habits, dailies, tags, and webhooks.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Description: "Habitica user ID (UUID). Can also be set via HABITICA_USER_ID environment variable.",
				Optional:    true,
			},
			"api_token": schema.StringAttribute{
				Description: "Habitica API token. Can also be set via HABITICA_API_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"client_author_id": schema.StringAttribute{
				Description: "Your Habitica user ID for the x-client header. Can also be set via HABITICA_CLIENT_AUTHOR_ID environment variable.",
				Optional:    true,
			},
			"client_app_name": schema.StringAttribute{
				Description: "Application name for the x-client header. Defaults to 'TerraformHabitica'.",
				Optional:    true,
			},
			"rate_limit_buffer": schema.Int64Attribute{
				Description: "Number of remaining requests at which to pause and wait for rate limit reset. Defaults to 5.",
				Optional:    true,
			},
		},
	}
}

func (p *HabiticaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config HabiticaProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get values from config or environment
	userID := getConfigOrEnv(config.UserID, "HABITICA_USER_ID")
	apiToken := getConfigOrEnv(config.APIToken, "HABITICA_API_TOKEN")
	clientAuthorID := getConfigOrEnv(config.ClientAuthorID, "HABITICA_CLIENT_AUTHOR_ID")

	if userID == "" {
		resp.Diagnostics.AddError(
			"Missing User ID",
			"The provider requires a user_id to be set in the configuration or via the HABITICA_USER_ID environment variable.",
		)
	}

	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"The provider requires an api_token to be set in the configuration or via the HABITICA_API_TOKEN environment variable.",
		)
	}

	if clientAuthorID == "" {
		resp.Diagnostics.AddError(
			"Missing Client Author ID",
			"The provider requires a client_author_id to be set in the configuration or via the HABITICA_CLIENT_AUTHOR_ID environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	clientAppName := ""
	if !config.ClientAppName.IsNull() {
		clientAppName = config.ClientAppName.ValueString()
	}

	rateLimitBuffer := 0
	if !config.RateLimitBuffer.IsNull() {
		rateLimitBuffer = int(config.RateLimitBuffer.ValueInt64())
	}

	c := client.New(client.Config{
		UserID:          userID,
		APIKey:          apiToken,
		ClientAuthorID:  clientAuthorID,
		ClientAppName:   clientAppName,
		RateLimitBuffer: rateLimitBuffer,
		BaseRetryDelay:  2 * time.Second,
	})

	resp.DataSourceData = c
	resp.ResourceData = c
}

func getConfigOrEnv(configValue types.String, envVar string) string {
	if !configValue.IsNull() && configValue.ValueString() != "" {
		return configValue.ValueString()
	}
	return os.Getenv(envVar)
}

func (p *HabiticaProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		tag.NewResource,
		habit.NewResource,
		daily.NewResource,
		webhook.NewResource,
	}
}

func (p *HabiticaProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
