terraform {
  required_providers {
    crud = {
      source = "registry.terraform.io/customcrud/customcrud"
    }
  }
}

provider "crud" {}

resource "crud" "file" {
  hooks {
    create = "../crud/create.sh"
    read   = "../crud/read.sh"
    update = "../crud/update.sh"
    delete = "../crud/delete.sh"
  }

  input = {
    content = "Hello, World tast!"
  }
}
