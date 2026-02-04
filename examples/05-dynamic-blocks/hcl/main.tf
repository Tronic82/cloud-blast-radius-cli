# Dynamic Blocks Example - HCL Mode (Limited)
# HCL mode cannot expand for_each and count blocks

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
}

variable "project_id" {
  type    = string
  default = "my-project"
}

# Example 1: for_each with map
variable "developers" {
  description = "Map of developer names to emails"
  type        = map(string)
  default = {
    "alice"   = "alice@example.com"
    "bob"     = "bob@example.com"
    "charlie" = "charlie@example.com"
  }
}

resource "google_project_iam_member" "dev_viewer" {
  for_each = var.developers
  
  project = var.project_id
  role    = "roles/viewer"
  member  = "user:${each.value}"
}

# Example 2: count with list
variable "service_accounts" {
  description = "List of service accounts"
  type        = list(string)
  default = [
    "deployer-sa",
    "backup-sa",
    "monitoring-sa"
  ]
}

resource "google_project_iam_member" "sa_editor" {
  count = length(var.service_accounts)
  
  project = var.project_id
  role    = "roles/editor"
  member  = "serviceAccount:${var.service_accounts[count.index]}@project.iam.gserviceaccount.com"
}

# Example 3: for_each with set
variable "environments" {
  description = "Set of environments"
  type        = set(string)
  default     = ["dev", "staging", "prod"]
}

resource "google_project_iam_member" "env_deployer" {
  for_each = var.environments
  
  project = "${each.key}-project"
  role    = "roles/editor"
  member  = "serviceAccount:deployer@project.iam.gserviceaccount.com"
}
