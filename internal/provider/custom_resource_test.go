// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccExampleResource(t *testing.T) {
	// Get absolute paths to the CRUD scripts
	crudDir := filepath.Join("..", "..", "crud")
	createScript := filepath.Join(crudDir, "create.sh")
	readScript := filepath.Join(crudDir, "read.sh")
	updateScript := filepath.Join(crudDir, "update.sh")
	deleteScript := filepath.Join(crudDir, "delete.sh")

	// Create import ID
	importID := map[string]interface{}{
		"create_script": []string{createScript},
		"read_script":   []string{readScript},
		"update_script": []string{updateScript},
		"delete_script": []string{deleteScript},
		"output": map[string]string{
			"id":       "test.txt",
			"filename": "test.txt",
			"content":  "Hello, World!",
		},
	}
	importIDBytes, err := json.Marshal(importID)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, "test.txt", "Initial content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("customcrud_resource.test", "output.filename", "test.txt"),
					resource.TestCheckResourceAttr("customcrud_resource.test", "output.content", "Initial content"),
				),
			},
			// Update testing
			{
				Config: testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, "test.txt", "Updated content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("customcrud_resource.test", "output.filename", "test.txt"),
					resource.TestCheckResourceAttr("customcrud_resource.test", "output.content", "Updated content"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "customcrud_resource.test",
				ImportState:                          true,
				ImportStateId:                        string(importIDBytes),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "output.id",
				ImportStateVerifyIgnore: []string{
					"input",
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccExampleResourceConfig(createScript, readScript, updateScript, deleteScript, filename, content string) string {
	return fmt.Sprintf(`
resource "customcrud_resource" "test" {
  create_script = [%[1]q]
  read_script  = [%[2]q]
  update_script = [%[3]q]
  delete_script = [%[4]q]
  input = {
    filename = %[5]q
    content = %[6]q
  }
}
`, createScript, readScript, updateScript, deleteScript, filename, content)
}
