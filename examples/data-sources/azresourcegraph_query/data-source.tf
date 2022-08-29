// Get all Resource IDs of all resources

data "azresourcegraph_query" "all_resource_ids" {
  query = "Resources | project id"
}

output "all_resource_ids" {
  value = [
    for obj in jsondecode(data.azresourcegraph_query.all_resource_ids.result) : obj.id
  ]
}

// Get all Key vaults with a specific tag in the `DEMO`
// Management group

data "azresourcegraph_query" "key_vaults" {
  management_group_ids = ["DEMO"]
  query                = <<QUERY
Resources
| where type =~ 'microsoft.keyvault/vaults'
| where tags.MY_TAG=~'my-value'
| project id, name, subscriptionId
QUERY
}
