# Policy Role Restrictions Example - Plan Mode

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

# Compliant bindings
resource "google_project_iam_member" "dev_viewer" {
  project = var.project_id
  role    = "roles/viewer"
  member  = "group:developers@example.com"
}

# Violation: developers with editor
resource "google_project_iam_member" "dev_editor" {
  project = var.project_id
  role    = "roles/editor"
  member  = "group:developers@example.com"
}

resource "google_bigquery_dataset_iam_member" "dev_bq_viewer" {
  dataset_id = "analytics"
  role       = "roles/bigquery.dataViewer"
  member     = "group:developers@example.com"
}

# Violation: service account owner
resource "google_project_iam_member" "sa_owner" {
  project = var.project_id
  role    = "roles/owner"
  member  = "serviceAccount:app-sa@production-project.iam.gserviceaccount.com"
}

# Violation: owner on prod
resource "google_project_iam_member" "prod_owner" {
  project = "prod-project"
  role    = "roles/owner"
  member  = "user:alice@example.com"
}

# Compliant: platform team owner
resource "google_project_iam_member" "platform_prod_owner" {
  project = "prod-project"
  role    = "roles/owner"
  member  = "group:platform-team@example.com"
}
