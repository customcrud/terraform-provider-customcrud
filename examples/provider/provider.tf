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
}
