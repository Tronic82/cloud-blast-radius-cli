# Variable Resolution Example - Plan Mode

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

variable "environment" {
  type    = string
  default = "prod"
}

variable "team_email" {
  type    = string
  default = "example.com"
}

resource "google_project_iam_member" "alice_viewer" {
  project = var.project_id
  role    = "roles/viewer"
  member  = "user:alice@${var.team_email}"
}

resource "google_service_account_iam_member" "bob_sa_user" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/app-sa@${var.project_id}.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountUser"
  member             = "user:bob@${var.team_email}"
}

resource "google_storage_bucket_iam_member" "deployer_bucket_access" {
  bucket = "${var.environment}-data-bucket"
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:deployer@${var.project_id}.iam.gserviceaccount.com"
}

resource "google_bigquery_dataset_iam_member" "analyst_dataset_access" {
  dataset_id = "${var.environment}_analytics_${var.project_id}"
  role       = "roles/bigquery.dataViewer"
  member     = "group:analysts@${var.team_email}"
}

resource "google_project_iam_member" "service_account_editor" {
  project = "${var.environment}-${var.project_id}"
  role    = "roles/editor"
  member  = "serviceAccount:${var.environment}-sa@${var.project_id}.iam.gserviceaccount.com"
}
