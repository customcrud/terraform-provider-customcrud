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
	// Create a temp file for the test
	tempFile, err := os.CreateTemp(t.TempDir(), "customcrud-ephemeral-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	content := "Hello from ephemeral resource!"
	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Use the same read script as data source - it outputs JSON with content
	openScript := "../../examples/file/hooks/read.sh"

	config := fmt.Sprintf(`
ephemeral "customcrud" "test" {
  hooks {
    open = %q
  }
  input = {
    path = %q
  }
}
`, openScript, tempFile.Name())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				// Ephemeral resources don't persist state, so this just validates
				// that the configuration is valid and the Open hook executes successfully
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
	openScript := "test_ephemeral/open.sh"

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
				// Success is enough, we verified complex input works in provider logs
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
					// Success is enough; serialization confirmed in logs
				},
			},
		})
	})
}
