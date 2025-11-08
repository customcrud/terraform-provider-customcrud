resource "customcrud" "file" {
  hooks {
    create = "${path.module}/hooks/create.sh"
    read   = "${path.module}/hooks/read.sh"
    update = "${path.module}/hooks/update.sh"
    delete = "${path.module}/hooks/delete.sh"
  }

  input = {
    content = "Hello, World!"
  }
}

data "customcrud" "file" {
  hooks {
    read = "${path.module}/hooks/read.sh"
  }
  input = {
    path = "/etc/hosts"
  }
}
