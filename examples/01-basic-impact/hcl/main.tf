# Basic Impact Analysis Example - HCL Mode
# This example demonstrates direct IAM access analysis

# Project-level IAM binding
resource "google_project_iam_member" "alice_viewer" {
  project = var.project_id
  role    = "roles/viewer"
  member  = "user:alice@example.com"
}

# BigQuery dataset IAM binding
resource "google_bigquery_dataset_iam_member" "alice_bq_viewer" {
  dataset_id = "analytics-dataset"
  role       = "roles/bigquery.dataViewer"
  member     = "user:alice@example.com"
}

# Storage bucket IAM binding
resource "google_storage_bucket_iam_member" "backup_sa_storage" {
  bucket = "data-bucket"
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:backup-sa@project.iam.gserviceaccount.com"
}

# Service account IAM binding
resource "google_service_account_iam_member" "bob_sa_user" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/app-sa@${var.project_id}.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountUser"
  member             = "user:bob@example.com"
}

# Additional example: Multiple members on same resource
resource "google_project_iam_member" "charlie_editor" {
  project = var.project_id
  role    = "roles/editor"
  member  = "user:charlie@example.com"
}

resource "google_project_iam_member" "dev_group_viewer" {
  project = var.project_id
  role    = "roles/viewer"
  member  = "group:developers@example.com"
}
