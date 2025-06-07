terraform {
  required_providers {
    customcrud = {
      source = "registry.terraform.io/customcrud/customcrud"
    }
  }
}

provider "customcrud" {}

resource "customcrud_resource" "file" {
  hooks {
    create = "../crud/create.sh"
    read   = "../crud/read.sh"
    update = "../crud/update.sh"
    delete = "../crud/delete.sh"
  }

  input = {
    content  = "Hello, World test!"
  }
}
