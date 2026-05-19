# Cloud Firewall attached to the gemma4 LLM Linode (172.238.48.187).
#
# This firewall is imported from the existing manually-created
# akamai-non-prod-3 firewall (id 5918188). All existing inbound rules
# are reproduced here so `terraform plan` against the import shows
# only the NEW rules for ports 80/443 needed by Let's Encrypt + Zuplo.
#
# To apply changes:
#   1. export LINODE_TOKEN=<PAT>
#   2. terraform plan
#   3. terraform apply

resource "linode_firewall" "gemma4" {
  label           = "akamai-non-prod-3"
  inbound_policy  = "DROP"
  outbound_policy = "ACCEPT"

  # Allow ICMP (ping) from anywhere — useful for liveness checks.
  inbound {
    label    = "accept-inbound-icmp"
    action   = "ACCEPT"
    protocol = "ICMP"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  # SSH proxy hosts — TCP 22/443 from known operator jump boxes.
  inbound {
    label    = "accept-inbound-tcp-ssh-proxies"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "22, 443"
    ipv4 = [
      "172.236.119.4/30",
      "172.234.160.4/30",
      "172.236.94.4/30",
      "206.83.109.247/32",
      "115.165.146.69/32",
      "206.83.106.54/32",
      "206.83.106.249/32",
      "206.83.106.242/32",
    ]
    ipv6 = [
      "2600:3c06::f03c:94ff:febe:162f/128",
      "2600:3c06::f03c:94ff:febe:16ff/128",
      "2600:3c06::f03c:94ff:febe:16c5/128",
      "2600:3c07::f03c:94ff:febe:16e6/128",
      "2600:3c07::f03c:94ff:febe:168c/128",
      "2600:3c07::f03c:94ff:febe:16de/128",
      "2600:3c08::f03c:94ff:febe:16e9/128",
      "2600:3c08::f03c:94ff:febe:1655/128",
      "2600:3c08::f03c:94ff:febe:16fd/128",
    ]
  }

  # EAA (Enterprise Application Access) proxy hosts — TCP 22/443.
  inbound {
    label    = "accept-inbound-tcp-eaa-proxies"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "22,443"
    ipv4 = [
      "139.144.212.168/31",
      "172.232.23.164/31",
    ]
  }

  # Port 8000 — direct access to the LLM (legacy). Kept enabled while
  # we transition to Caddy + HTTPS, but Zuplo now goes through 443.
  # Sources: Cloudflare (Zuplo) + Akamai infrastructure ranges.
  inbound {
    label    = "accept-inbound-8000"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "8000"
    ipv4 = [
      "172.232.0.0/13",
      "172.104.0.0/15",
      "139.144.0.0/14",
      "45.33.0.0/16",
      "45.76.0.0/14",
      "206.83.109.247/32",
      "206.83.106.54/32",
      "206.83.106.249/32",
      "104.64.128.44/32",
      "104.64.128.46/32",
      "104.64.128.47/32",
      "173.245.48.0/20",
      "103.21.244.0/22",
      "103.22.200.0/22",
      "141.101.64.0/18",
      "108.162.192.0/18",
      "190.93.240.0/20",
      "197.234.240.0/22",
      "198.41.128.0/17",
      "162.158.0.0/15",
      "104.16.0.0/13",
      "104.24.0.0/14",
      "172.64.0.0/13",
      "131.0.72.0/22",
    ]
    ipv6 = [
      "2400:cb00::/32",
      "2606:4700::/32",
      "2803:f800::/32",
      "2405:b500::/32",
      "2405:8100::/32",
      "2a06:98c0::/29",
      "2c0f:f248::/32",
    ]
  }

  # NEW: Port 80 from anywhere — needed for Let's Encrypt HTTP-01
  # challenge so Caddy can auto-issue/renew TLS certs.
  inbound {
    label    = "accept-inbound-http-letsencrypt"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "80"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  # NEW: Port 443 from anywhere — main HTTPS entrypoint for the
  # LLM via Caddy reverse-proxy. Zuplo (Cloudflare Workers) reaches
  # the LLM through this, but we leave it open since it's TLS-secured
  # and Caddy proxies all unknown hosts to a 404.
  inbound {
    label    = "accept-inbound-https"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "443"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  # NOTE: device attachment (Linode 94564094 / gemma4) is NOT managed
  # here — it was attached manually before import and Terraform would
  # try to detach/reattach which is disruptive. Add a `linodes = [...]`
  # block here once we're confident; for now the FW-rule-only diff is
  # what we want.
}
