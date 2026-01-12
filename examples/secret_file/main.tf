terraform {
  required_providers {
    customcrud = {
      source = "registry.terraform.io/customcrud/customcrud"
    }
    google = {
      source = "registry.terraform.io/hashicorp/google"
    }
  }
}

provider "google" {
  project = local.project
}

locals {
  project                 = "myproject"
  password_secret_name    = "test"
  password_secret_version = element(split("/", data.customcrud.latest_version.output.name), -1)
}

data "customcrud" "latest_version" {
  hooks {
    read = "hooks/read_secret_version.sh"
  }

  input = {
    secret_name = local.password_secret_name
    project     = local.project
  }
}

ephemeral "google_secret_manager_secret_version" "password_version" {
  secret  = local.password_secret_name
  version = local.password_secret_version
}

resource "customcrud" "secret_file" {
  hooks {
    create = "../../internal/provider/test_write_only/create.sh"
    read   = "../../internal/provider/test_write_only/read.sh"
    update = "../../internal/provider/test_write_only/update.sh"
    delete = "../../internal/provider/test_write_only/delete.sh"
  }

  input_wo = jsonencode({
    content = ephemeral.google_secret_manager_secret_version.password_version.secret_data
  })

  input = {
    // Force an update when the password changes (ephemeral resources don't appear in plans)
    secret_version = local.password_secret_version
  }
}
