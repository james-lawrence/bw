variable "project" {
  description = "project ID to provision within"
}

variable "deployment_fqdn" {
  description = "dns name used to deploy"
}

variable "acme-email" {
  description = "email use to register with lets encrypt"
}

variable "region" {
  description = "data center region to provision within"
  default     = "us-east1"
}

variable "zone" {
  description = "data center to provision within"
  default     = "us-east1-b"
}

variable "image" {
  description = "base image to use"
  default     = "ubuntu-1910-eoan-v20191217"
}

variable "cluster" {
  description = "name of the cluster"
  default     = "example"
}

variable "dns-managed-zone" {
  description = "dns managed zone for creating dns records"
}

provider "google" {
  project = var.project
  region  = var.region
}

# output "instances" {
#   value = module.dht.instances
# }
#
# output "private-ips" {
#   value = module.dht.private
# }
#
# output "endpoint" {
#   value = module.dht.endpoint
# }
