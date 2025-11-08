package provider

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomCrudDataSource_File(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "customcrud-ds-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	content := "Hello from data source!"
	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	readScript := "../../examples/modules/file/hooks/read.sh"

	config := strings.ReplaceAll(`
	data "customcrud" "test" {
	  hooks {
	    read = "%READ_SCRIPT%"
	  }
	  input = {
	    path = "%TEMP_PATH%"
	  }
	}

	output "ds_content" {
	  value = data.customcrud.test.output.content
	}
	`, "%READ_SCRIPT%", readScript)
	config = strings.ReplaceAll(config, "%TEMP_PATH%", tempFile.Name())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("ds_content", content),
				),
			},
		},
	})
}
