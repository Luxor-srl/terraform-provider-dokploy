# Look up a project by id
data "dokploy_project" "by_id" {
  id = "abc123"
}

# Look up a project by name
data "dokploy_project" "by_name" {
  name = "My Project"
}

output "project_environments" {
  value = data.dokploy_project.by_name.environments
}
