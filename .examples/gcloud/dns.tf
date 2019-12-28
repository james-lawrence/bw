data "google_dns_managed_zone" "root" {
  name = var.dns-managed-zone
}

resource "google_compute_address" "default" {
  name = "${var.cluster}-endpoint"
}

resource "google_compute_forwarding_rule" "default" {
  name                  = "${var.cluster}-bearded-wookie-discovery"
  load_balancing_scheme = "EXTERNAL"
  ip_protocol           = "TCP"
  ip_address            = google_compute_address.default.address
  port_range            = "2001"
  target                = google_compute_target_pool.default.self_link
}

resource "google_compute_forwarding_rule" "agent" {
  name                  = "${var.cluster}-bearded-wookie-agent"
  load_balancing_scheme = "EXTERNAL"
  ip_protocol           = "TCP"
  ip_address            = google_compute_address.default.address
  port_range            = "2002"
  target                = google_compute_target_pool.default.self_link
}

resource "google_dns_record_set" "deploy" {
  name = "deploy.${data.google_dns_managed_zone.root.dns_name}"
  type = "A"
  ttl  = 15

  managed_zone = data.google_dns_managed_zone.root.name
  rrdatas      = [google_compute_address.default.address]
}
