// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccExampleResource(t *testing.T) {
	content := "Initial content"
	updatedContent := "Updated content"

	createScript := "../../crud/create.sh"
	readScript := "../../crud/read.sh"
	updateScript := "../../crud/update.sh"
	deleteScript := "../../crud/delete.sh"

	// Single test case with all steps including import
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create testing
			{
				Config: testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, content),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("customcrud_resource.test", "output.content", content),
					resource.TestCheckResourceAttrSet("customcrud_resource.test", "id"),
				),
			},
			// Import testing - this should happen after create but before update
			{
				Config:                  testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, content),
				ResourceName:            "customcrud_resource.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccResourceImportStateIdFunc("customcrud_resource.test", createScript, readScript, updateScript, deleteScript),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"hooks", "input"},
			},
			// Update testing
			{
				Config: testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, updatedContent),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("customcrud_resource.test", "output.content", updatedContent),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// Helper function to generate import state ID.
func testAccResourceImportStateIdFunc(resourceName, createScript, readScript, updateScript, deleteScript string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("resource ID not set")
		}

		// Format: create_cmd,read_cmd,update_cmd,delete_cmd:id
		return fmt.Sprintf("%s,%s,%s,%s:%s",
			createScript, readScript, updateScript, deleteScript, rs.Primary.ID), nil
	}
}

func testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, content string) string {
	return fmt.Sprintf(`
resource "customcrud_resource" "test" {
  hooks {
    create = %[1]q
    read   = %[2]q
    update = %[3]q
    delete = %[4]q
  }
  input = {
    content = %[5]q
  }
}
`, createScript, readScript, updateScript, deleteScript, content)
}
