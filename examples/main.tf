terraform {
  required_providers {
    customcrud = {
      source = "registry.terraform.io/customcrud/customcrud"
    }
  }
}

provider "customcrud" {}

resource "customcrud_resource" "file" {
  create_script = ["./crud/create.sh"]
  read_script   = ["./crud/read.sh"]
  update_script = ["./crud/update.sh"]
  delete_script = ["./crud/delete.sh"]

  input = {
    content  = "Hello, World!"
    filename = "example.txt"
  }
}
