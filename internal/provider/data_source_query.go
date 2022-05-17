package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/resourcegraph/mgmt/2021-03-01/resourcegraph"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceQuery() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for querying resources managed by Azure Resource Manager.",

		ReadContext: dataSourceQueryRead,

		Schema: map[string]*schema.Schema{
			"query": {
				Description: "The query to execute.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"subscription_ids": {
				Description:   `Azure subscription ids against which to execute the query.`,
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"management_group_ids"},
				MinItems:      1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"management_group_ids": {
				Description:   `Azure management groups against which to execute the query.`,
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"subscription_ids"},
				MinItems:      1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"result": {
				Description: `The queries output in raw json format.`,
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceQueryRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*clients).resourceGraph
	query := d.Get("query").(string)

	opts := resourcegraph.QueryRequestOptions{
		ResultFormat: resourcegraph.ResultFormatObjectArray,
	}

	queryRequest := resourcegraph.QueryRequest{
		Options: &opts,
		Query:   &query,
	}

	if _, ok := d.GetOk("subscription_ids"); ok {
		subscriptions := d.Get("subscription_ids").(*schema.Set).List()
		subs := make([]string, len(subscriptions))
		for i, v := range subscriptions {
			subs[i] = v.(string)
		}
		queryRequest.Subscriptions = &subs
	}

	if _, ok := d.GetOk("management_group_ids"); ok {
		managementgroups := d.Get("management_group_ids").(*schema.Set).List()
		grps := make([]string, len(managementgroups))
		for i, v := range managementgroups {
			grps[i] = v.(string)
		}
		queryRequest.ManagementGroups = &grps
	}

	data, err := doResourceQuery(ctx, client, queryRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return diag.Errorf("Failed to marshal query results to JSON string: %v", err)
	}

	d.SetId(uuid.New().String())
	d.Set("result", string(jsonData))

	return nil
}

func doResourceQuery(ctx context.Context, client *resourcegraph.BaseClient, queryRequest resourcegraph.QueryRequest) (data interface{}, err error) {
	var results []interface{}
	resp, err := client.Resources(ctx, queryRequest)
	if err != nil {
		return nil, fmt.Errorf("Query failed: %v", err)
	}
	results = append(results, resp.Data)

	if resp.SkipToken != nil {
		skipToken := resp.SkipToken
		for {
			deltaQueryRequest := queryRequest
			deltaQueryRequest.Options.SkipToken = skipToken
			deltaResp, err := client.Resources(ctx, queryRequest)
			if err != nil {
				return nil, fmt.Errorf("Delta query failed: %v", err)
			}
			results = append(results, deltaResp.Data)

			if deltaResp.SkipToken == nil {
				break
			}

			skipToken = deltaResp.SkipToken
		}
	}

	return results, nil
}
