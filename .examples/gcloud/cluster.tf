data "google_compute_image" "boot" {
  name    = var.image
  project = "ubuntu-os-cloud"
}

resource "google_compute_target_pool" "default" {
  name             = "${var.cluster}-pool"
  session_affinity = "CLIENT_IP"
}

resource "google_project_iam_custom_role" "default" {
  role_id     = "${var.cluster}_role"
  title       = "${var.cluster} role"
  description = "provides the permissions necessary to run bearded-wookie"
  permissions = [
    "dns.changes.create",
    "dns.changes.get",
    "dns.changes.list",
    "dns.managedZones.get",
    "dns.managedZones.list",
    "dns.resourceRecordSets.create",
    "dns.resourceRecordSets.delete",
    "dns.resourceRecordSets.list",
    "dns.resourceRecordSets.update",
  ]
}

resource "google_service_account" "default" {
  account_id   = "${var.cluster}-account"
  display_name = "${var.cluster} service account"
}

resource "google_service_account_iam_binding" "default" {
  service_account_id = google_service_account.default.name
  role               = google_project_iam_custom_role.default.id
  members            = []
}


resource "google_compute_instance_template" "default" {
  lifecycle {
    create_before_destroy = true
  }

  name_prefix = var.cluster
  tags        = []
  description = "this template is used to create dht server instances."

  labels = {
    environment = "${terraform.workspace}"
  }

  instance_description = ""
  machine_type         = "f1-micro"

  scheduling {
    automatic_restart   = true
    on_host_maintenance = "MIGRATE"
  }

  disk {
    source_image = data.google_compute_image.boot.self_link
    auto_delete  = true
    boot         = true
  }

  network_interface {
    network = "default"
    access_config {}
  }

  service_account {
    email = google_service_account.default.email
    scopes = [
      "cloud-platform",
      "https://www.googleapis.com/auth/ndev.clouddns.readwrite",
    ]
  }

  metadata = {
    user-data = data.template_file.cloudinit.rendered
  }
}

resource "google_compute_instance_group_manager" "default" {
  name = "igm-${var.cluster}"

  base_instance_name = var.cluster
  zone               = var.zone
  target_size        = 1
  target_pools       = ["${google_compute_target_pool.default.self_link}"]

  named_port {
    name = "bearded-wookie-discovery"
    port = 2001
  }

  version {
    name              = "node"
    instance_template = google_compute_instance_template.default.self_link
  }

  version {
    name              = "leader"
    instance_template = google_compute_instance_template.default.self_link

    target_size {
      fixed = 1
    }
  }

  update_policy {
    type                  = "PROACTIVE"
    minimal_action        = "REPLACE"
    max_surge_fixed       = 1
    max_unavailable_fixed = 0
    min_ready_sec         = 60
  }
}

output "pool" {
  value = google_compute_target_pool.default.self_link
}
