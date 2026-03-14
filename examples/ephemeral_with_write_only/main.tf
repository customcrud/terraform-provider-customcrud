terraform {
  required_providers {
    customcrud = {
      source = "registry.terraform.io/customcrud/customcrud"
    }
  }
}

# Read a secret from a file using an ephemeral resource. Ephemeral resources are
# not stored in state, making them ideal for sensitive data like passwords and
# API keys.
ephemeral "customcrud" "urandom" {
  hooks {
    # Open is called on every terraform run, if this is an expensive operation,
    # you'll need to implement a cache within the hook using your inputs as keys
    open = "hooks/open.sh"
  }
}

# Use the secret via input_wo (write-only) so it flows to the create/update
# hooks but is never persisted in Terraform state.
resource "customcrud" "file" {
  hooks {
    create = "hooks/create.sh"
    read   = "hooks/read.sh"
    # For an example of how to get a new urandom value each time your file
    # inputs are updated, comment out the update hook
    update = "hooks/update.sh"
    delete = "hooks/delete.sh"
  }

  input = {
    path = "secret.txt"
  }

  input_wo = jsonencode({
    content = ephemeral.customcrud.urandom.output.content
  })
}
