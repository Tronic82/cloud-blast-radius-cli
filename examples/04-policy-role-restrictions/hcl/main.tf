# Policy Role Restrictions Example - HCL Mode

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

# ✅ COMPLIANT: Developers with allowed viewer role
resource "google_project_iam_member" "dev_viewer" {
  project = var.project_id
  role    = "roles/viewer"
  member  = "group:developers@example.com"
}

# ❌ VIOLATION: Developers with denied editor role
resource "google_project_iam_member" "dev_editor" {
  project = var.project_id
  role    = "roles/editor"
  member  = "group:developers@example.com"
}

# ✅ COMPLIANT: Developers with allowed BigQuery viewer
resource "google_bigquery_dataset_iam_member" "dev_bq_viewer" {
  dataset_id = "analytics"
  role       = "roles/bigquery.dataViewer"
  member     = "group:developers@example.com"
}

# ❌ VIOLATION: Service account with owner role
resource "google_project_iam_member" "sa_owner" {
  project = var.project_id
  role    = "roles/owner"
  member  = "serviceAccount:app-sa@project.iam.gserviceaccount.com"
}

# ❌ VIOLATION: Owner on production resource
resource "google_project_iam_member" "prod_owner" {
  project = "prod-project"
  role    = "roles/owner"
  member  = "user:alice@example.com"
}

# ✅ COMPLIANT: Platform team can be owner on prod
resource "google_project_iam_member" "platform_prod_owner" {
  project = "prod-project"
  role    = "roles/owner"
  member  = "group:platform-team@example.com"
}
