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

  # `default_inputs` will be merged into the input field sent to all hooks.
  # They will not show up in any plans as they are only merged at execution
  # time, however they will still be visible in debug logs if TF_LOG is enabled.
  default_inputs = {
    api_url = var.api_url
  }

  # `sensitive_default_inputs` works like `default_inputs` but masks values in
  # all log output and error diagnostics. Use this for secrets (API keys,
  # tokens, etc.) that should never appear in debug logs. Sensitive defaults
  # take priority over `default_inputs` when keys overlap; resource-level
  # input takes priority over both.
  sensitive_default_inputs = {
    api_key = var.api_key
  }
}
