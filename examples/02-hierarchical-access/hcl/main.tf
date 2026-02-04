# Hierarchical Access Example - HCL Mode
# Demonstrates project-level roles granting access to ALL resources of a type

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
  description = "GCP Project ID"
  type        = string
  default     = "my-project"
}

# Project-level BigQuery viewer role
# This grants access to ALL BigQuery datasets in the project
resource "google_project_iam_member" "alice_bq_viewer" {
  project = var.project_id
  role    = "roles/bigquery.dataViewer"
  member  = "user:alice@example.com"
}

# Project-level Storage viewer role
# This grants access to ALL Storage buckets in the project
resource "google_project_iam_member" "bob_storage_viewer" {
  project = var.project_id
  role    = "roles/storage.objectViewer"
  member  = "user:bob@example.com"
}

# Project-level BigQuery editor role
# This grants write access to ALL BigQuery datasets
resource "google_project_iam_member" "charlie_bq_editor" {
  project = var.project_id
  role    = "roles/bigquery.dataEditor"
  member  = "user:charlie@example.com"
}

# Project-level Storage admin role
# This grants admin access to ALL Storage buckets
resource "google_project_iam_member" "dave_storage_admin" {
  project = var.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:backup-sa@project.iam.gserviceaccount.com"
}

# Example: Specific resource binding (NOT hierarchical)
# This only grants access to this specific dataset
resource "google_bigquery_dataset_iam_member" "eve_specific_dataset" {
  dataset_id = "public-analytics"
  role       = "roles/bigquery.dataViewer"
  member     = "user:eve@example.com"
}

# The datasets that Alice, Bob, Charlie, and Dave can access
# (These are just examples to show what resources exist)
resource "google_bigquery_dataset" "customer_data" {
  dataset_id = "customer_data"
  location   = "US"
}

resource "google_bigquery_dataset" "analytics" {
  dataset_id = "analytics"
  location   = "US"
}

resource "google_storage_bucket" "data_lake" {
  name     = "data-lake-bucket"
  location = "US"
}

resource "google_storage_bucket" "backups" {
  name     = "backup-bucket"
  location = "US"
}
