# Secret File provider

In this example:
- how to use the write-only inputs (we use the secret version number so that we
  can invalidate and update the file when the secret value changes
  automatically.)
- using customcrud's data source when appropriate terraform provider resources
  aren't available (in this case the google secret manager provider has several
  data resources for secret versions, but they all store the secret value in
  state, which we're trying to avoid with the ephemeral resource)
