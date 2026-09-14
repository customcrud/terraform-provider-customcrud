terraform {
  required_providers {
    customcrud = {
      source = "registry.terraform.io/customcrud/customcrud"
    }
  }
}

provider "customcrud" {
  # Optionally limit how many executions run in parallel.
  # This is useful if your scripts or tools are not safe to run concurrently.
  # This option defaults to 0 (unlimited parallelism).
  parallelism = 1

  # merged into the input field sent to the all hooks. They will not show up in
  # any plans as they are only merged at execution time
  default_inputs = {
    api_url = var.api_url
  }

  # `sensitive_default_inputs` behaves the same as `default_inputs`, but values
  # are masked in debug logs and error output.
  sensitive_default_inputs = {
    api_key = var.api_key
  }
}

variable "api_url" {
  type = string
}

variable "api_key" {
  type      = string
  sensitive = true
}

resource "customcrud" "defaults" {
  hooks {
    create = <<-EOF
      bash -c 'jq --arg id "$(uuidgen)" "del(.input) + {id: \$id}"'
    EOF
    read   = "jq .output"
    delete = "cat"
  }
}