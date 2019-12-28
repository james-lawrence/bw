resource "google_compute_firewall" "default" {
  name    = "${var.cluster}-bearded-wookie"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["2001", "2002", "2003", "2004"] // DISCOVERY, RPC, SWIM, RAFT.
  }

  allow {
    protocol = "udp"
    ports    = ["2003", "2004"] // SWIM, TORRENT.
  }

  source_ranges = ["10.0.0.0/8"]
}

resource "google_compute_firewall" "discovery" {
  name    = "${var.cluster}-bearded-wookie-discovery"
  network = "default"

  # IMPORTANT: in real world deployments only the discovery service should be exposed.
  # bearded-wookie assumes its running inside of a VPN.
  allow {
    protocol = "tcp"
    ports    = ["2001", "2002"] // DISCOVERY, RPC
  }

  source_ranges = ["0.0.0.0/0"]
}
