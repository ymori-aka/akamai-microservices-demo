# LKE cluster for the microservices-demo, region-parametrized (var.region).
#
# This is the Tokyo (jp-tyo-3) target of the Osaka→Tokyo consolidation. It is a
# NEW cluster that runs in parallel with the existing click-ops jp-osa cluster
# (id 598466); the jp-osa cluster is left untouched until cut-over is verified,
# then decommissioned manually. Mirrors the current cluster: HA control plane,
# 3× g6-dedicated-4, k8s 1.35.
#
# After apply, fetch the kubeconfig with:
#   terraform output -raw lke_kubeconfig | base64 --decode > kubeconfig-tyo.yaml
#   KUBECONFIG=kubeconfig-tyo.yaml kubectl get nodes

resource "linode_lke_cluster" "demo" {
  label       = var.lke_label
  region      = var.region
  k8s_version = var.lke_k8s_version
  tags        = ["microservices-demo", "tokyo-migration"]

  control_plane {
    high_availability = var.lke_ha_control_plane
  }

  # Node pool count is managed here (no autoscaler), matching the current
  # cluster's fixed 3-node pool.
  pool {
    type  = var.lke_node_type
    count = var.lke_node_count
  }
}

output "lke_id" {
  value = linode_lke_cluster.demo.id
}

output "lke_status" {
  value = linode_lke_cluster.demo.status
}

output "lke_api_endpoints" {
  value = linode_lke_cluster.demo.api_endpoints
}

output "lke_kubeconfig" {
  description = "Base64-encoded kubeconfig. Decode with `base64 --decode`."
  value       = linode_lke_cluster.demo.kubeconfig
  sensitive   = true
}
