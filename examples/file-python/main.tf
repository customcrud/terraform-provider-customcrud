terraform {
  required_providers {
    customcrud = {
      source = "registry.terraform.io/customcrud/customcrud"
    }
  }
}

resource "customcrud" "file" {
  hooks {
    create = "poetry run file-provider --action=create"
    read   = "poetry run file-provider --action=read"
    update = "poetry run file-provider --action=update"
    delete = "poetry run file-provider --action=delete"
  }

  input = {
    content = "Hello from Python!"
  }
}

output "file_contents" {
  value = customcrud.file.output.content
}

output "file_id" {
  value = customcrud.file.id
}
