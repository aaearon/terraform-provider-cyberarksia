// Package provider implements the CyberArk SIA Terraform provider
package provider

import (
	"context"
	"os"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/client"
	"github.com/cyberark/ark-sdk-golang/pkg/services/identity"
	"github.com/cyberark/ark-sdk-golang/pkg/services/sia"
	"github.com/cyberark/ark-sdk-golang/pkg/services/uap"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces
var _ provider.Provider = &CyberArkSIAProvider{}

// CyberArkSIAProvider defines the provider implementation
type CyberArkSIAProvider struct {
	// version is set to the provider version on release
	version string
}

// CyberArkSIAProviderModel describes the provider data model
type CyberArkSIAProviderModel struct {
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	IdentityURL types.String `tfsdk:"identity_url"`
}

// ProviderData holds the ARK SDK instances shared with resources
// This struct is passed to resources via resp.ResourceData in Configure()
type ProviderData struct {
	// AuthContext holds authentication state including in-memory profile
	// This prevents filesystem profile loading and keyring caching
	AuthContext *client.ISPAuthContext

	// SIAAPI provides access to SIA WorkspacesDB() and SecretsDB() APIs
	SIAAPI *sia.ArkSIAAPI

	// UAPClient provides access to UAP Db() API for policy management
	UAPClient *uap.ArkUAPAPI

	// IdentityClient provides access to Identity UsersService() and DirectoriesService() for principal lookups
	IdentityClient *identity.ArkIdentityAPI

	// CertificatesClient handles certificate CRUD operations
	// Initialized on-demand by certificate resource Configure()
	CertificatesClient *client.CertificatesClient

	// VMService provides access to VM policy CRUD operations
	VMService interface{}
}

// New is a helper function to simplify provider server and testing implementation
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CyberArkSIAProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name
func (p *CyberArkSIAProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cyberarksia"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data
func (p *CyberArkSIAProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for CyberArk Secure Infrastructure Access (SIA). " +
			"Manage database workspaces, secrets, certificates, and access policies for Zero Standing Privilege (ZSP) " +
			"and Just-In-Time (JIT) database access. Supports 60+ database engines including AWS RDS, Azure SQL Database, " +
			"PostgreSQL, MySQL, MongoDB, Oracle, SQL Server, and more. " +
			"Uses the CyberArk ARK SDK for OAuth2 authentication with automatic token refresh.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description: "Service account username in full format (e.g., 'my-service-account@cyberark.cloud.12345'). " +
					"The tenant information is automatically extracted from the username by the ARK SDK. " +
					"Can also be set via CYBERARK_USERNAME environment variable.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"password": schema.StringAttribute{
				Description: "Service account password. " +
					"Can also be set via CYBERARK_PASSWORD environment variable.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"identity_url": schema.StringAttribute{
				Description: "CyberArk Identity tenant URL (e.g., https://abc123.cyberark.cloud). " +
					"OPTIONAL - only needed for GovCloud (https://abc123.cyberarkgov.cloud) or custom identity deployments. " +
					"If not provided, the URL is automatically resolved from the username by the ARK SDK. " +
					"Can also be set via CYBERARK_IDENTITY_URL environment variable.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

// Configure prepares a CyberArk SIA API client for data sources and resources
func (p *CyberArkSIAProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config CyberArkSIAProviderModel

	// Read configuration from Terraform
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get values from environment variables if not set in configuration
	username := getEnvOrConfig(config.Username.ValueString(), EnvUsername)
	password := getEnvOrConfig(config.Password.ValueString(), EnvPassword)
	identityURL := getEnvOrConfig(config.IdentityURL.ValueString(), EnvIdentityURL)

	// Validate required fields
	if username == "" {
		resp.Diagnostics.AddError(
			"Missing username",
			"username must be set in provider configuration or via CYBERARK_USERNAME environment variable",
		)
	}
	if password == "" {
		resp.Diagnostics.AddError(
			"Missing password",
			"password must be set in provider configuration or via CYBERARK_PASSWORD environment variable",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Log configuration (without sensitive data)
	LogProviderConfig(ctx, &config)

	// Initialize authentication with in-memory profile (bypasses ~/.ark_cache and ~/.ark/profiles/)
	// Uses IdentityServiceUser method - SDK auto-resolves Identity URL from username if not provided
	LogAuthStart(ctx)
	authCtx, err := client.NewISPAuth(ctx, &client.AuthConfig{
		Username:    username,
		Password:    password,
		IdentityURL: identityURL, // Optional - SDK auto-resolves from username
	})
	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "provider configuration"))
		return
	}
	LogAuthSuccess(ctx)

	// Initialize SIA API client
	// Returns *sia.ArkSIAAPI for WorkspacesDB() and SecretsDB() access
	LogSIAClientInit(ctx)
	siaAPI, err := client.NewSIAClient(ctx, authCtx)
	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "SIA client initialization"))
		return
	}
	LogSIAClientSuccess(ctx)

	// Initialize UAP API client
	// Returns *uap.ArkUAPAPI for Db() policy management access
	LogUAPClientInit(ctx)
	uapAPI, err := client.NewUAPClient(ctx, authCtx)
	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "UAP client initialization"))
		return
	}
	LogUAPClientSuccess(ctx)

	// Initialize Identity API client
	// Returns *identity.ArkIdentityAPI for UsersService() and DirectoriesService() access
	LogIdentityClientInit(ctx)
	identityAPI, err := client.NewIdentityClient(ctx, authCtx)
	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "Identity client initialization"))
		return
	}
	LogIdentityClientSuccess(ctx)

	// Initialize VM Service for VM access policy management
	// Retrieved from the UAPClient which is already initialized
	vmService := uapAPI.VM()

	// Create provider data for resource sharing
	providerData := &ProviderData{
		AuthContext:    authCtx,
		SIAAPI:         siaAPI,
		UAPClient:      uapAPI,
		IdentityClient: identityAPI,
		VMService:      vmService,
	}

	// Make provider data available to resources and data sources
	resp.ResourceData = providerData
	resp.DataSourceData = providerData
}

// Resources defines the resources implemented in the provider
func (p *CyberArkSIAProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDatabaseWorkspaceResource,
		NewDatabaseSecretResource,
		NewVMSecretResource,
		NewCertificateResource,
		NewDatabasePolicyResource,
		NewDatabasePolicyPrincipalAssignmentResource,
		NewDatabasePolicyWorkspaceAssignmentResource,
		NewTargetSetResource,
		NewVMPolicyResource,
		NewVMPolicyPrincipalAssignmentResource,
	}
}

// DataSources defines the data sources implemented in the provider
func (p *CyberArkSIAProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDatabasePolicyDataSource,
		NewPrincipalDataSource,
	}
}

// getEnvOrConfig returns config value if set, otherwise falls back to environment variable
func getEnvOrConfig(configValue string, envVar string) string {
	if configValue != "" {
		return configValue
	}
	return os.Getenv(envVar)
}
