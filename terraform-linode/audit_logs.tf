# Object Storage bucket + access key for Akamai Cloud Pulse audit logs.
#
# Audit log streams (audit_logs + lke_audit_logs) deliver JSON log
# objects here via a monitor "destination". A CronJob in LKE
# (kubernetes-manifests/monitoring/audit-log-ingester.yaml) pulls new
# objects and pushes them into Loki for Grafana.
#
# The streams + destination themselves are created by
# scripts/setup-audit-streams.sh (Monitor API — not yet in the Linode
# Terraform provider).

resource "linode_object_storage_bucket" "audit_logs" {
  region = var.region
  label  = "akamai-audit-logs"
}

# Scoped access key — read/write limited to the audit-logs bucket.
resource "linode_object_storage_key" "audit_logs" {
  label = "audit-logs-rw"
  bucket_access {
    bucket_name = linode_object_storage_bucket.audit_logs.label
    region      = var.region
    permissions = "read_write"
  }
}

output "audit_logs_bucket" {
  value = linode_object_storage_bucket.audit_logs.label
}

output "audit_logs_bucket_host" {
  value = "${linode_object_storage_bucket.audit_logs.label}.${var.obj_host_suffix}"
}

output "audit_logs_access_key" {
  value     = linode_object_storage_key.audit_logs.access_key
  sensitive = true
}

output "audit_logs_secret_key" {
  value     = linode_object_storage_key.audit_logs.secret_key
  sensitive = true
}
