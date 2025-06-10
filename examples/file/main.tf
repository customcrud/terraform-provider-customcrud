terraform {
  required_providers {
    customcrud = {
      source = "registry.terraform.io/customcrud/customcrud"
    }
  }
}

provider "customcrud" {}

resource "customcrud" "file" {
  hooks {
    create = "hooks/create.sh"
    read   = "hooks/read.sh"
    update = "hooks/update.sh"
    delete = "hooks/delete.sh"
  }

  input = {
    content = "Hello, World!"
  }
}
