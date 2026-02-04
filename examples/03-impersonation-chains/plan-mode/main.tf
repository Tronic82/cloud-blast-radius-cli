# Impersonation Chains Example - Plan Mode

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

# Simple impersonation chain
resource "google_project_iam_member" "dev_sa_viewer" {
  project = "dev-project"
  role    = "roles/viewer"
  member  = "serviceAccount:dev-sa@production-project.iam.gserviceaccount.com"
}

resource "google_service_account_iam_member" "dev_can_impersonate_prod" {
  service_account_id = "projects/production-project/serviceAccounts/prod-sa@production-project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:dev-sa@production-project.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "prod_sa_owner" {
  project = "production-project"
  role    = "roles/owner"
  member  = "serviceAccount:prod-sa@production-project.iam.gserviceaccount.com"
}

# Multi-hop chain
resource "google_service_account_iam_member" "alice_impersonate_deploy" {
  service_account_id = "projects/production-project/serviceAccounts/deploy-sa@production-project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountUser"
  member             = "user:alice@example.com"
}

resource "google_service_account_iam_member" "deploy_impersonate_admin" {
  service_account_id = "projects/production-project/serviceAccounts/admin-sa@production-project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:deploy-sa@production-project.iam.gserviceaccount.com"
}

resource "google_bigquery_dataset_iam_member" "admin_sa_bq_admin" {
  dataset_id = "sensitive-customer-data"
  role       = "roles/bigquery.admin"
  member     = "serviceAccount:admin-sa@production-project.iam.gserviceaccount.com"
}

resource "google_storage_bucket_iam_member" "admin_sa_storage_admin" {
  bucket = "production-secrets"
  role   = "roles/storage.admin"
  member = "serviceAccount:admin-sa@production-project.iam.gserviceaccount.com"
}
