# Linode Terraform provider.
# Token is supplied via environment variable LINODE_TOKEN
# (export LINODE_TOKEN=<your-PAT>) so the token never lives in the repo.

terraform {
  required_version = ">= 1.5"
  required_providers {
    linode = {
      source  = "linode/linode"
      version = "~> 2.31"
    }
  }
}

provider "linode" {
  # token read from LINODE_TOKEN env var
}
