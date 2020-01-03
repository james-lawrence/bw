data "template_file" "cloudinit" {
  template = <<EOF
#cloud-config
# setup package repositories.
apt:
  preserve_sources_list: true
  sources:
    beardedwookie:
      source: "ppa:jljatone/bw"

packages:
  - bearded-wookie

runcmd:
 - systemctl disable --now snapd.service snapd.socket
 - systemctl enable --now bearded-wookie.service bearded-wookie-notifications.service

write_files:
  - encoding: b64
    content: ${base64encode(data.template_file.bearded-wookie-config.rendered)}
    owner: root:root
    path: /etc/bearded-wookie/default/agent.config
    permissions: '0644'
  - encoding: b64
    content: ${base64encode(file("bearded-wookie-agent.env"))}
    owner: root:root
    path: /etc/bearded-wookie/default/agent.env
    permissions: '0600'
EOF
}
