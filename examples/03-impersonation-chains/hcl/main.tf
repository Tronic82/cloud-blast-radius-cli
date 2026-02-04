# Impersonation Chains Example - HCL Mode
# Demonstrates service account impersonation and privilege escalation

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

# ============================================================================
# EXAMPLE 1: Simple Impersonation Chain (2-hop)
# ============================================================================

# dev-sa has minimal direct access
resource "google_project_iam_member" "dev_sa_viewer" {
  project = "dev-project"
  role    = "roles/viewer"
  member  = "serviceAccount:dev-sa@project.iam.gserviceaccount.com"
}

# dev-sa can impersonate prod-sa
resource "google_service_account_iam_member" "dev_can_impersonate_prod" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/prod-sa@project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:dev-sa@project.iam.gserviceaccount.com"
}

# prod-sa has powerful access
resource "google_project_iam_member" "prod_sa_owner" {
  project = "prod-project"
  role    = "roles/owner"
  member  = "serviceAccount:prod-sa@project.iam.gserviceaccount.com"
}

# ============================================================================
# EXAMPLE 2: Multi-Hop Impersonation Chain (3-hop)
# ============================================================================

# Alice has no direct resource access
# But Alice can impersonate deploy-sa
resource "google_service_account_iam_member" "alice_impersonate_deploy" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/deploy-sa@project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountUser"
  member             = "user:alice@example.com"
}

# deploy-sa can impersonate admin-sa
resource "google_service_account_iam_member" "deploy_impersonate_admin" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/admin-sa@project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:deploy-sa@project.iam.gserviceaccount.com"
}

# admin-sa has access to sensitive data
resource "google_bigquery_dataset_iam_member" "admin_sa_bq_admin" {
  dataset_id = "sensitive-customer-data"
  role       = "roles/bigquery.admin"
  member     = "serviceAccount:admin-sa@project.iam.gserviceaccount.com"
}

resource "google_storage_bucket_iam_member" "admin_sa_storage_admin" {
  bucket = "production-secrets"
  role   = "roles/storage.admin"
  member = "serviceAccount:admin-sa@project.iam.gserviceaccount.com"
}

# ============================================================================
# EXAMPLE 3: Circular Impersonation (Detected and Prevented)
# ============================================================================

# sa-a can impersonate sa-b
resource "google_service_account_iam_member" "a_impersonate_b" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/sa-b@project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:sa-a@project.iam.gserviceaccount.com"
}

# sa-b can impersonate sa-c
resource "google_service_account_iam_member" "b_impersonate_c" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/sa-c@project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:sa-b@project.iam.gserviceaccount.com"
}

# sa-c can impersonate sa-a (creates a cycle)
# Blast Radius detects this and prevents infinite loops
resource "google_service_account_iam_member" "c_impersonate_a" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/sa-a@project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:sa-c@project.iam.gserviceaccount.com"
}

# sa-c has access to resources
resource "google_project_iam_member" "sa_c_editor" {
  project = var.project_id
  role    = "roles/editor"
  member  = "serviceAccount:sa-c@project.iam.gserviceaccount.com"
}
