variable "project" {
  description = "project ID to provision within"
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

variable "cluster_size" {
  description = "number of servers to build"
  default     = 3
}

variable "dns-managed-zone" {
  description = "dns managed zone for creating dns records"
}

provider "google" {
  project = var.project
  region  = var.region
}
