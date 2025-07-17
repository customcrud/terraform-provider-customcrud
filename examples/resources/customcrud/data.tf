# Example: customcrud data source

data "customcrud" "file" {
  hooks {
    read = "file/read.sh"
  }

  input = {
    id = "some-file-id"
  }
}

output "file_content" {
  value = data.customcrud.file.output["content"]
} 