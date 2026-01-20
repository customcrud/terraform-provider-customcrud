package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
