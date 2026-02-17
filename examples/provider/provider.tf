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

  # `default_inputs` can be used for secret and non-secret config alike, and
  # will be merged into the input field sent to the all hooks. They will not
  # show up in any plans as they are only merged at execution time, this allows
  # you to pass in secrets via this method, as they will not be stored in any
  # plans or state, it will however still be visible in any debug logs if TF_LOG
  # is enabled so proceed with caution.
  default_inputs = {
    api_url = var.api_url
    api_key = var.api_key
  }
}
