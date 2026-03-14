package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCustomCrudEphemeral_Basic(t *testing.T) {
	openScript := "../../examples/ephemeral_with_write_only/hooks/open.sh"

	config := fmt.Sprintf(`
ephemeral "customcrud" "test" {
  hooks {
    open = %q
  }
}
`, openScript)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
		},
	})
}

func TestAccCustomCrudEphemeral_WithWriteOnly(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "secret.txt")

	openScript := "../../examples/ephemeral_with_write_only/hooks/open.sh"
	createScript := "../../examples/ephemeral_with_write_only/hooks/create.sh"
	readScript := "../../examples/ephemeral_with_write_only/hooks/read.sh"
	updateScript := "../../examples/ephemeral_with_write_only/hooks/update.sh"
	deleteScript := "../../examples/ephemeral_with_write_only/hooks/delete.sh"

	config := fmt.Sprintf(`
ephemeral "customcrud" "urandom" {
  hooks {
    open = %q
  }
}

resource "customcrud" "file" {
  hooks {
    create = %q
    read   = %q
    update = %q
    delete = %q
  }

  input = {
    path = %q
  }

  input_wo = jsonencode({
    content = ephemeral.customcrud.urandom.output.content
  })
}
`, openScript, createScript, readScript, updateScript, deleteScript, secretFile)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						content, err := os.ReadFile(secretFile)
						if err != nil {
							return fmt.Errorf("secret file was not created: %w", err)
						}
						if len(content) == 0 {
							return fmt.Errorf("secret file is empty, expected urandom content")
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccCustomCrudEphemeral_AllHooks(t *testing.T) {
	markerFile := filepath.Join(t.TempDir(), "marker.txt")
	openScript := "test_ephemeral/open.sh"
	renewScript := "test_ephemeral/renew.sh"
	closeScript := "test_ephemeral/close.sh"

	config := fmt.Sprintf(`
ephemeral "customcrud" "all" {
  hooks {
    open  = %q
    renew = %q
    close = %q
  }
  input = {
    name        = "test-all-hooks"
    marker_file = %q
  }
}
`, openScript, renewScript, closeScript, markerFile)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						// Verify marker file was written by open script
						content, err := os.ReadFile(markerFile)
						if err != nil {
							return fmt.Errorf("failed to read marker file: %w", err)
						}
						// open script writes "open"
						if !strings.Contains(string(content), "open") {
							return fmt.Errorf("marker file does not contain 'open', got: %q", string(content))
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccCustomCrudEphemeral_OpenFailure(t *testing.T) {
	openScript := "test_ephemeral_failures/open.sh"

	config := fmt.Sprintf(`
ephemeral "customcrud" "fail" {
  hooks {
    open = %q
  }
  input = {
    test = "failure"
  }
}
`, openScript)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ExpectError: regexp.MustCompile(
					`(?s)Error: Open Script Failed.*` +
						`script execution failed with exit code 13: exit status 13.*` +
						`Exit Code: 13.*` +
						`Stdout:.*` +
						`Stderr: Failed to open ephemeral resource: Access denied.*`),
			},
		},
	})
}

func TestAccCustomCrudEphemeral_ComplexInput(t *testing.T) {
	openScript := "../../examples/ephemeral_with_write_only/hooks/open.sh"

	config := fmt.Sprintf(`
ephemeral "customcrud" "complex" {
  hooks {
    open = %q
  }
  input = {
    name = "complex-test"
    metadata = {
      nested = {
        key = "value"
        list = [1, 2, 3]
      }
    }
  }
}
`, openScript)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
		},
	})
}

func TestAccCustomCrudEphemeral_Parallelism(t *testing.T) {
	dir, err := filepath.Abs("test_parallel")
	if err != nil {
		t.Fatalf("Failed to resolve test_parallel dir: %v", err)
	}
	openScript := filepath.Join(dir, "create.sh") // Use create.sh as it has locking logic

	t.Run("parallelism=1 passes", func(t *testing.T) {
		config := fmt.Sprintf(`
provider "customcrud" {
  parallelism = 1
}
ephemeral "customcrud" "p1" {
  count = 2
  hooks {
    open = %q
  }
  input = { name = "ephemeral_parallel_1" }
}
`, openScript)
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
				},
			},
		})
	})
}
