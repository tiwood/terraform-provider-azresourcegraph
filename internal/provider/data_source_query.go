package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
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

	format := armresourcegraph.ResultFormatObjectArray
	opts := armresourcegraph.QueryRequestOptions{
		ResultFormat: &format,
	}

	queryRequest := armresourcegraph.QueryRequest{
		Options: &opts,
		Query:   &query,
	}

	if _, ok := d.GetOk("subscription_ids"); ok {
		subscriptions := d.Get("subscription_ids").(*schema.Set).List()
		subs := make([]*string, len(subscriptions))
		for i, v := range subscriptions {
			sub := v.(string)
			subs[i] = &sub
		}
		queryRequest.Subscriptions = subs
	}

	if _, ok := d.GetOk("management_group_ids"); ok {
		managementgroups := d.Get("management_group_ids").(*schema.Set).List()
		grps := make([]*string, len(managementgroups))
		for i, v := range managementgroups {
			grp := v.(string)
			grps[i] = &grp
		}
		queryRequest.ManagementGroups = grps
	}

	data, err := doResourceQuery(ctx, &client, queryRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return diag.Errorf("failed to marshal query results to JSON string: %v", err)
	}

	d.SetId(uuid.New().String())
	err = d.Set("result", string(jsonData))
	if err != nil {
		return diag.Errorf("failed to set query result: %v", err)
	}

	return nil
}

func doResourceQuery(ctx context.Context, client *armresourcegraph.Client, queryRequest armresourcegraph.QueryRequest) (data interface{}, err error) {
	var results []interface{}

	for {
		resp, err := client.Resources(ctx, queryRequest, nil)
		if err != nil {
			return nil, fmt.Errorf("query failed: %v", err)
		}
		if resp.Count != nil && *resp.Count > 0 && resp.Data != nil {
			results = append(results, (resp.Data.([]interface{}))...)
		}
		if resp.SkipToken == nil {
			break
		}
		queryRequest.Options.SkipToken = resp.SkipToken
	}

	return results, nil
}
