resource "google_compute_firewall" "default" {
  name    = "${var.cluster}-bearded-wookie"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["2000"] // agent
  }

  allow {
    protocol = "udp"
    ports    = ["2000"]
  }

  source_ranges = ["10.0.0.0/8"]
}

// bearded-wookie best practice is to run it inside of a VPN.
resource "google_compute_firewall" "insecure" {
  name    = "${var.cluster}-bearded-wookie-discovery"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["443", "80"]
  }

  source_ranges = ["0.0.0.0/0"]
}
