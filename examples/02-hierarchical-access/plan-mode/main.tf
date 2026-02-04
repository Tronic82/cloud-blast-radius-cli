# Hierarchical Access Example - Plan Mode

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
  default     = "production-project"
}

# Project-level BigQuery viewer role
resource "google_project_iam_member" "alice_bq_viewer" {
  project = var.project_id
  role    = "roles/bigquery.dataViewer"
  member  = "user:alice@example.com"
}

# Project-level Storage viewer role
resource "google_project_iam_member" "bob_storage_viewer" {
  project = var.project_id
  role    = "roles/storage.objectViewer"
  member  = "user:bob@example.com"
}

# Project-level BigQuery editor role
resource "google_project_iam_member" "charlie_bq_editor" {
  project = var.project_id
  role    = "roles/bigquery.dataEditor"
  member  = "user:charlie@example.com"
}

# Project-level Storage admin role
resource "google_project_iam_member" "dave_storage_admin" {
  project = var.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:backup-sa@production-project.iam.gserviceaccount.com"
}

# Example resources that are affected by hierarchical access
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
