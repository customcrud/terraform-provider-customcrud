// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccExampleResource(t *testing.T) {
	content := "Initial content"
	updatedContent := "Updated content"

	createScript := "../../examples/crud/create.sh"
	readScript := "../../examples/crud/read.sh"
	updateScript := "../../examples/crud/update.sh"
	deleteScript := "../../examples/crud/delete.sh"

	// Single test case with all steps including import
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create testing
			{
				Config: testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, content),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("customcrud.test", "output.content", content),
					resource.TestCheckResourceAttrSet("customcrud.test", "id"),
				),
			},
			// Import testing - this should happen after create but before update
			{
				Config:                  testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, content),
				ResourceName:            "customcrud.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccResourceImportStateIdFunc("customcrud.test", createScript, readScript, updateScript, deleteScript),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"hooks", "input"},
			},
			// Update testing
			{
				Config: testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, updatedContent),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("customcrud.test", "output.content", updatedContent),
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

		importData := importStateData{
			Id: rs.Primary.ID,
			Hooks: map[string]string{
				"create": createScript,
				"read":   readScript,
				"update": updateScript,
				"delete": deleteScript,
			},
		}

		// Get input from state if it exists
		if input, ok := rs.Primary.Attributes["input"]; ok {
			var inputMap map[string]interface{}
			if err := json.Unmarshal([]byte(input), &inputMap); err == nil {
				importData.Input = inputMap
			}
		}

		// Get output from state if it exists
		if output, ok := rs.Primary.Attributes["output"]; ok {
			var outputMap map[string]interface{}
			if err := json.Unmarshal([]byte(output), &outputMap); err == nil {
				importData.Output = outputMap
			}
		}

		jsonData, err := json.Marshal(importData)
		if err != nil {
			return "", fmt.Errorf("failed to marshal import data: %v", err)
		}

		return string(jsonData), nil
	}
}

func testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, content string) string {
	return fmt.Sprintf(`
resource "customcrud" "test" {
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
