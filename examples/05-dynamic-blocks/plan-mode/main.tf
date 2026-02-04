# Dynamic Blocks Example - Plan Mode (Fully Expanded)

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
  default = "production-project"
}

variable "developers" {
  type = map(string)
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

variable "service_accounts" {
  type = list(string)
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
  member  = "serviceAccount:${var.service_accounts[count.index]}@production-project.iam.gserviceaccount.com"
}

variable "environments" {
  type    = set(string)
  default = ["dev", "staging", "prod"]
}

resource "google_project_iam_member" "env_deployer" {
  for_each = var.environments
  
  project = "${each.key}-project"
  role    = "roles/editor"
  member  = "serviceAccount:deployer@production-project.iam.gserviceaccount.com"
}
