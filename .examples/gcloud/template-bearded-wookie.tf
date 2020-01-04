data "template_file" "bearded-wookie-config" {
  template = file("bearded-wookie-agent-config.yml")
  vars = {
    acme_email            = var.acme-email
    bearded_wookie_server = "${trimsuffix(google_dns_record_set.deploy.name, ".")}"
  }
}

data "template_file" "bearded-wookie-local-config" {
  template = file("bearded-wookie-client-config.yml")
  vars = {
    bearded_wookie_server = "${trimsuffix(google_dns_record_set.deploy.name, ".")}"
  }
}

resource "local_file" "bearded-wookie-config" {
  content         = data.template_file.bearded-wookie-local-config.rendered
  filename        = "../.bwconfig/example"
  file_permission = "0600"
}
