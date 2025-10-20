terraform {
  required_providers {
    customcrud = {
      source = "registry.terraform.io/customcrud/customcrud"
    }
  }
}

provider "customcrud" {}

resource "customcrud" "test" {
  hooks {
    create = "./create.sh"
    read   = "./read.sh"
    delete = "./delete.sh"
  }

  input = {
    b = {
      c = ["a", "b", "c"]
      d = []
    }
  }
}