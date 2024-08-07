data "template_file" "cloudinit" {
  template = <<EOF
#cloud-config
# setup package repositories.
apt:
  preserve_sources_list: true
  sources:
    beardedwookie:
      source: "ppa:jljatone/bw"
    golang:
      source: "ppa:longsleep/golang-backports"

packages:
  - zip
  - bearded-wookie
  - nginx-full
  - golang-1.16

runcmd:
 - systemctl disable --now snapd.service snapd.socket
 - unzip -d /var/lib/bearded-wookie-example/filesystem-overlay /var/lib/bearded-wookie-example/filesystem-overlay.zip
 - rsync --recursive --progress --checksum /var/lib/bearded-wookie-example/filesystem-overlay/ /
 - echo systemctl enable --now swapfile.service swapfile.swap
 - systemctl enable --now bearded-wookie.service bearded-wookie-notifications.service
 - systemctl restart nginx.service

write_files:
  - encoding: b64
    content: ${filebase64(data.archive_file.filesystem.output_path)}
    owner: root:root
    path: /var/lib/bearded-wookie-example/filesystem-overlay.zip
    permissions: '0600'
EOF
}
