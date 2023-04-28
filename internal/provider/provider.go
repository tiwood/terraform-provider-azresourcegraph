package provider

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resourcegraph/mgmt/2021-03-01/resourcegraph"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var diags diag.Diagnostics

func init() {
	schema.DescriptionKind = schema.StringMarkdown
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"tenant_id": {
					Description: "The Tenant ID which should be used. This can also be sourced from the `AZRGRAPH_TENANT_ID` Environment Variable.",
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("AZRGRAPH_TENANT_ID", nil),
				},
				"client_id": {
					Description: "The Client ID which should be used. This can also be sourced from the `AZRGRAPH_CLIENT_ID` Environment Variable.",
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("AZRGRAPH_CLIENT_ID", nil),
				},
				"client_secret": {
					Description: "The Client Secret which should be used. This can also be sourced from the `AZRGRAPH_CLIENT_SECRET` Environment Variable.",
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("AZRGRAPH_CLIENT_SECRET", nil),
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"azresourcegraph_query": dataSourceQuery(),
			},
			ResourcesMap: map[string]*schema.Resource{},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

type clients struct {
	resourceGraph *resourcegraph.BaseClient
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var resource = "https://management.azure.com"

		tenantID := d.Get("tenant_id").(string)
		clientID := d.Get("client_id").(string)
		clientSecret := d.Get("client_secret").(string)

		client := resourcegraph.New()
		client.PollingDelay = 20
		client.RetryAttempts = 20
		client.RetryDuration = 5 * time.Second

		if clientID == "" || clientSecret == "" || tenantID == "" {
			authorizer, err := auth.NewAuthorizerFromCLIWithResource(resource)
			if err != nil {
				diags = append(diags, diag.Errorf("Failed to create authorizer from CLI: %v", err)...)
				return nil, diags
			}
			client.Authorizer = authorizer
		} else {
			clientCredentialCfg := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
			clientCredentialCfg.Resource = resource
			authorizer, err := clientCredentialCfg.Authorizer()

			if err != nil {
				diags = append(diags, diag.Errorf("Failed to create authorizer with credentials: %v", err)...)
				return nil, diags
			}
			client.Authorizer = authorizer
		}

		return &clients{
			resourceGraph: &client,
		}, nil
	}
}
