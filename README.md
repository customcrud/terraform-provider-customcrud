# Custom CRUD Terraform Provider

This Terraform provider enables the execution of custom scripts for Create, Read, Update, and Delete (CRUD) operations. It's designed to bridge the gap between Terraform and existing automation scripts or custom workflows.

## Features

- Execute custom scripts for resource lifecycle management
- Pass input parameters to scripts as JSON
- Capture script output as resource attributes
- Support for all CRUD operations with flexible script configuration

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23

## Building The Provider

1. Clone the repository
2. Enter the repository directory
3. Build the provider using the Go `install` command:

```shell
go install
```

## Using the Provider

The provider allows you to define resources that use custom scripts for their lifecycle operations. Here's a basic example:

```hcl
resource "customcrud_resource" "example" {
  hooks {
    create = "./scripts/create.sh"
    read   = "./scripts/read.sh"
    update = "./scripts/update.sh"
    delete = "./scripts/delete.sh"
  }

  input = {
    key = "value"
    # Additional input parameters
  }
}
```

### Script Requirements

Your scripts should:
1. Accept JSON input via stdin
2. Return JSON output to stdout
3. Use appropriate exit codes (0 for success, non-zero for failure)
4. Handle the specific CRUD operation they're designed for

### Input/Output Format

Scripts receive input as JSON:
```json
{
  "id": "resource-id",  
  "input": {
    "key": "value",
    ...
  },
  "output": {
    "key": "value",
    ...
  }
}
```

Scripts should return output as JSON:
```json
{
  "key": "value",
  "id": "resource-id",
  ...
}
```

The `id` field is required in the output of the create script and will be used to track the resource. The output from scripts will be stored in the resource's `output` attribute and can be referenced in other resources.

## Development

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.
