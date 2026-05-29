# Shared variables for the Linode infra.
#
# `region` is intentionally a variable so we never get region-locked again:
# the whole stack moves by changing one value. Default is jp-tyo-3 (Tokyo 3),
# the only JP region that has BOTH:
#   - ACLP Logs capability (Cloud Pulse Logs), absent in jp-osa, and
#   - E3 Object Storage endpoint (Cloud Pulse Object Storage metrics);
#     jp-osa is E1, which does NOT support Object Storage Cloud Pulse.
# (Tokyo 2 / ap-northeast has no Object Storage at all — not a valid target.)
variable "region" {
  description = "Linode region for the stack (jp-tyo-3 = Tokyo 3, E3 Object Storage + ACLP Logs)."
  type        = string
  default     = "jp-tyo-3"
}

# Object Storage S3 host suffix for the region. NOTE: jp-tyo-3 buckets resolve
# under jp-tyo-1.linodeobjects.com (not jp-tyo-3). Keep this in sync with
# `linode-cli object-storage endpoints` if the region changes.
variable "obj_host_suffix" {
  description = "Object Storage S3 endpoint host suffix for the region."
  type        = string
  default     = "jp-tyo-1.linodeobjects.com"
}

# --- LKE cluster sizing (mirrors the current jp-osa cluster) ---
variable "lke_label" {
  description = "LKE cluster label."
  type        = string
  default     = "microservices-demo-tyo"
}

variable "lke_k8s_version" {
  description = "Kubernetes version for the LKE cluster."
  type        = string
  default     = "1.35"
}

variable "lke_node_type" {
  description = "Linode plan for LKE worker nodes."
  type        = string
  default     = "g6-dedicated-4"
}

variable "lke_node_count" {
  description = "Number of LKE worker nodes."
  type        = number
  default     = 3
}

variable "lke_ha_control_plane" {
  description = "Enable the HA control plane (matches current cluster)."
  type        = bool
  default     = true
}

# Control Plane ACL: the ONLY IPs allowed to reach the k8s API server.
#   [0] this admin machine, [1] mgmt-server, [2] self-hosted runner egress.
# Keep the runner IP or deploy-tokyo.yml (runs on that runner) is locked out.
variable "cp_acl_ipv4" {
  description = "IPv4 CIDRs allowed to reach the LKE control plane (admin machine, mgmt-server, CI runner)."
  type        = list(string)
  default = [
    "115.165.146.69/32", # this admin machine
    "172.234.87.163/32", # mgmt-server
    "206.83.106.54/32",  # self-hosted GitHub Actions runner egress
  ]
}
