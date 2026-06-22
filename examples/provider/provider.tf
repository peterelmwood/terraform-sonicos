terraform {
  required_providers {
    sonicos = {
      source = "peterelmwood/sonicos"
    }
  }
}

# Credentials can be set here or via the SONICOS_HOST / SONICOS_USERNAME /
# SONICOS_PASSWORD environment variables. Prefer environment variables or a
# secrets manager for the password rather than committing it.
provider "sonicos" {
  host     = "https://192.0.2.1:8443"
  username = "tf-automation"
  password = var.sonicos_password

  # Most appliances use a self-signed certificate by default.
  insecure = true

  # Activate the pending configuration after each write (default). Set to false
  # to stage changes and commit them yourself for a single-transaction apply.
  commit_on_apply = true
}

variable "sonicos_password" {
  type      = string
  sensitive = true
}
