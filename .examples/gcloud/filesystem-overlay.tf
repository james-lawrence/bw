resource "local_file" "bearded-wookie-client-config" {
  content         = templatefile("${path.module}/.templates/bearded-wookie-client-config.yml", {
    bearded_wookie_server = trimsuffix(google_dns_record_set.deploy.name, ".")
  })
  filename        = "../.bwconfig/example"
  file_permission = "0600"
}

resource "local_file" "bearded-wookie-agent-config" {
  content  = templatefile("${path.module}/.templates/bearded-wookie-agent-config.yml", {
    acme_email            = var.acme-email
    bearded_wookie_server = trimsuffix(google_dns_record_set.deploy.name, ".")
  })
  filename        = "${path.module}/.filesystem/etc/bearded-wookie/default/agent.config"
  file_permission = "0600"
}

resource "local_file" "bearded-wookie-authorization" {
  content         = file(pathexpand("~/.config/bearded-wookie/private.key.pub"))
  filename        = "${path.module}/.filesystem/etc/bearded-wookie/default/bw.auth.keys"
  file_permission = "0600"
}

data "archive_file" "filesystem" {
  type        = "zip"
  output_path = "${path.module}/.dist/filesystem.archive.zip"
  source_dir  = "${path.module}/.filesystem/"

  depends_on = [
    local_file.bearded-wookie-agent-config,
    local_file.bearded-wookie-client-config,
    local_file.bearded-wookie-authorization,
  ]
}