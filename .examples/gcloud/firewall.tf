resource "google_compute_firewall" "default" {
  name    = "${var.cluster}-bearded-wookie"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["2001", "2002", "2003", "2004", "2006"] // DISCOVERY, RPC, SWIM, RAFT, AUTOCERT.
  }

  allow {
    protocol = "udp"
    ports    = ["2003", "2005"] // SWIM, TORRENT.
  }

  source_ranges = ["10.0.0.0/8"]
}

resource "google_compute_firewall" "discovery" {
  name    = "${var.cluster}-bearded-wookie-discovery"
  network = "default"

  # IMPORTANT: in real world deployments only the discovery service (port 2001) should be exposed.
  # bearded-wookie assumes its running inside of a VPN. simply remove port 2002 from this list.
  allow {
    protocol = "tcp"
    ports    = ["2001", "2002"] // DISCOVERY, RPC
  }

  source_ranges = ["0.0.0.0/0"]
}
