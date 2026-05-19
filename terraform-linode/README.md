# terraform-linode

Declarative configuration for Linode (Akamai Cloud) resources used by
this demo. Initially scoped to **Cloud Firewalls** — other resources
(LKE cluster, NodeBalancer, Managed PG, Object Storage, Linode VMs)
can be imported here over time.

## Prerequisites

- `terraform >= 1.5`
- A Linode Personal Access Token with **Firewall: Read/Write**
  permissions (use the same `LINODE_PAT` GitHub Secret if you want a
  single token for everything)

## Usage

```bash
export LINODE_TOKEN=<your-PAT>

cd terraform-linode

# First run only: download the linode provider plugin.
terraform init

# Preview changes
terraform plan

# Apply
terraform apply
```

## Current resources

| Resource | Linode ID | Purpose |
|----------|-----------|---------|
| `linode_firewall.gemma4` | 5918188 | Cloud Firewall for the `gemma4` LLM VM (172.238.48.187). Allows SSH/EAA proxies, port 8000 from Cloudflare (legacy Zuplo path), and ports 80/443 from anywhere (Let's Encrypt + Caddy HTTPS). |

## Importing more resources

```bash
# 1. Write a resource block in a new .tf file.
# 2. Import the existing object.
terraform import linode_<resource_type>.<name> <linode_id>
# 3. Run `terraform plan` and tweak the .tf until the diff is empty.
```

## State

Currently **local state** (`terraform.tfstate`). For team use, migrate to
remote state in Linode Object Storage:

```hcl
terraform {
  backend "s3" {
    endpoint                    = "https://<region>.linodeobjects.com"
    bucket                      = "tfstate-akamai-demo"
    key                         = "linode/terraform.tfstate"
    region                      = "us-east-1"  # ignored, but required
    skip_credentials_validation = true
    skip_region_validation      = true
    skip_metadata_api_check     = true
  }
}
```
