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

data "customcrud" "file" {
  hooks {
    read = "hooks/read.sh"
  }
  input = {
    path = "/etc/hosts"
  }
}

output "file_contents" {
  value = data.customcrud.file.output.content
}
