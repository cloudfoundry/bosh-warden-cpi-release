resource "random_string" "account_suffix" {
  length  = 4
  upper   = false
  special = false
  lower   = true
  numeric  = true
}

resource "google_service_account" "service_account" {
  account_id = "${var.prefix}-sa-${random_string.account_suffix.result}"
}

resource "google_compute_address" "director" {
  name = "${var.prefix}-dir"
}

resource "google_compute_address" "director_internal" {
  name         = "${var.prefix}-dir-internal"
  address_type = "INTERNAL"
  subnetwork   = google_compute_subnetwork.subnetwork.self_link
}

resource "google_compute_address" "jumpbox_internal" {
  name         = "${var.prefix}-jumpbox-internal"
  address_type = "INTERNAL"
  subnetwork   = google_compute_subnetwork.subnetwork.self_link
}

resource "google_compute_address" "int_internal" {
  count        = 3
  name         = "${var.prefix}-int-internal-${count.index}"
  address_type = "INTERNAL"
  subnetwork   = google_compute_subnetwork.subnetwork.self_link
}

resource "google_compute_network" "manual" {
  name                    = "${var.prefix}-manual"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnetwork" {
  name          = var.prefix
  ip_cidr_range = var.google_subnetwork_range
  network       = google_compute_network.manual.self_link
  region        = var.google_region
}

resource "google_compute_firewall" "internal" {
  name        = "${var.prefix}-int"
  description = "BOSH CI Internal Traffic"
  network     = google_compute_network.manual.self_link
  source_tags = [var.google_firewall_internal]
  target_tags = [var.google_firewall_internal]

  allow {
    protocol = "tcp"
  }

  allow {
    protocol = "udp"
  }

  allow {
    protocol = "icmp"
  }
}

resource "google_compute_firewall" "external" {
  name        = "${var.prefix}-ext"
  description = "BOSH CI External Traffic"
  network     = google_compute_network.manual.self_link
  source_ranges = ["0.0.0.0/0"]
  target_tags = [var.google_firewall_external]

  allow {
    protocol = "tcp"
    ports    = ["22", "443", "4222", "6868", "25250", "25555", "25777"]
  }

  allow {
    protocol = "udp"
    ports    = ["53"]
  }

  allow {
    protocol = "icmp"
  }
}

resource "google_compute_address" "jumpbox" {
  name   = "${var.prefix}-jumpbox-ip"
  region = var.google_region
}


resource "google_compute_firewall" "mbus-jumpbox" {
  name    = "${var.prefix}-jumpbox-ingress"
  network = google_compute_network.manual.name

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["22", "6868"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["jumpbox"]
}

resource "google_compute_firewall" "director-ingress" {
  name    = "${var.prefix}-director-from-jumpbox"
  network = google_compute_network.manual.name

  allow {
    protocol = "tcp"
    ports    = ["22", "5001", "5985", "5986", "6868", "8443", "8844", "10006", "25555"]
  }

  source_tags = ["jumpbox"]
  target_tags   = ["bosh-deployed"]
}

resource "google_compute_firewall" "bosh-internal" {
  name    = "${var.prefix}-bosh-internal"
  network = google_compute_network.manual.name

  allow {
    protocol = "all"
  }

  source_tags = ["bosh-deployed"]
  target_tags   = ["bosh-deployed"]
}
