resource "customcrud" "file" {
  hooks {
    create = "file/create.sh"
    read   = "file/read.sh"
    update = "file/update.sh"
    delete = "file/delete.sh"
  }

  input = {
    content = "Hello, World!"
  }
}
