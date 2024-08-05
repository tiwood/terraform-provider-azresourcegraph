package provider

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
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
				"use_azure_default_credential": {
					Description: "Use Azure Default Credential for authentication. " +
						"If this is true, the provider will try to authenticate using the following mechanisms in order:" +
						"Environment variables > Workload identity > Managed Identity > Azure CLI > Azure Developer CLI " +
						"This can also be sourced from the `AZRGRAPH_USE_AZURE_DEFAULT_CREDENTIAL` Environment Variable. " +
						"Note, that the Client Secret flow will take precedence over the Azure Default Credential. Defaults to true.",
					Type:        schema.TypeBool,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("AZRGRAPH_USE_AZURE_DEFAULT_CREDENTIAL", true),
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
	resourceGraph armresourcegraph.Client
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {

		clientID := d.Get("client_id").(string)
		clientSecret := d.Get("client_secret").(string)
		tenantID := d.Get("tenant_id").(string)
		useDefaultCred := d.Get("use_azure_default_credential").(bool)

		var cred azcore.TokenCredential
		var err error

		if clientID != "" && clientSecret != "" && tenantID != "" {
			// Client secret credentials
			cred, err = azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
			if err != nil {
				diags = append(diags, diag.Errorf("unable to create ClientSecretCredential: %v", err)...)
				return nil, diags
			}
		} else if useDefaultCred {
			// Azure Default Credential
			cred, err = azidentity.NewDefaultAzureCredential(nil)
			if err != nil {
				diags = append(diags, diag.Errorf("unable to create DefaultAzureCredential: %v", err)...)
				return nil, diags
			}
		} else {
			diags = append(diags, diag.Errorf("no authenticaton credenials provided")...)
			return nil, diags
		}

		clientFactory, err := armresourcegraph.NewClientFactory(cred, nil)
		if err != nil {
			diags = append(diags, diag.Errorf("unable to create client factory: %v", err)...)
			return nil, diags
		}
		client := clientFactory.NewClient()

		return &clients{
			resourceGraph: *client,
		}, nil
	}
}
