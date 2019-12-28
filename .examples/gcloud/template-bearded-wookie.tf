data "template_file" "bearded-wookie-config" {
  template = file("bearded-wookie-agent-config.yml")
  vars = {
    bearded_wookie_server = "${var.deployment_fqdn}"
  }
}

data "template_file" "bearded-wookie-local-config" {
  template = file("bearded-wookie-client-config.yml")
  vars = {
    bearded_wookie_server = "${var.deployment_fqdn}"
  }
}

resource "local_file" "bearded-wookie-config" {
  content         = data.template_file.bearded-wookie-local-config.rendered
  filename        = ".bwconfig/example"
  file_permission = "0600"
}
