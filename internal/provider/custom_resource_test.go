// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccExampleResource(t *testing.T) {
	content := "Initial content"
	updatedContent := "Updated content"

	createScript := "../../examples/file/hooks/create.sh"
	readScript := "../../examples/file/hooks/read.sh"
	updateScript := "../../examples/file/hooks/update.sh"
	deleteScript := "../../examples/file/hooks/delete.sh"

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

func TestAccExampleResourceEdgeCases(t *testing.T) {
	createScript := "test_edgecases/create.sh"
	readScript := "test_edgecases/read.sh"
	deleteScript := "test_edgecases/delete.sh"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExampleResourceEdgeCaseConfig(createScript, readScript, deleteScript),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("customcrud.test", "output.a.#", "6"),
					resource.TestCheckResourceAttr("customcrud.test", "output.a.0", "1"),
					resource.TestCheckResourceAttr("customcrud.test", "output.a.1", "2"),
					resource.TestCheckResourceAttr("customcrud.test", "output.a.2", "false"),
					// resource.TestCheckResourceAttr("customcrud.test", "output.a.3", ""), // null value can't be checked directly

					resource.TestCheckResourceAttr("customcrud.test", "output.a.4.0.b", "3"),
					resource.TestCheckResourceAttr("customcrud.test", "output.a.5.#", "3"),
					resource.TestCheckResourceAttr("customcrud.test", "output.a.5.0", "1"),
					resource.TestCheckResourceAttr("customcrud.test", "output.a.5.1", "2"),
					resource.TestCheckResourceAttr("customcrud.test", "output.a.5.2", "3"),
				),
				// jq -n '{id: 1, a: [1, "2", false, null, [{"b": 3}], [1, 2, 3]]}'
			},
			// Import testing
			{
				Config:                  testAccExampleResourceEdgeCaseConfig(createScript, readScript, deleteScript),
				ResourceName:            "customcrud.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccResourceImportStateIdFunc("customcrud.test", createScript, readScript, "", deleteScript),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"hooks"},
			},
		},
	})
}

func TestAccResourceScriptFailures(t *testing.T) {
	createScript := "test_failures/create.sh"
	readScript := "test_failures/read.sh"
	deleteScript := "test_failures/delete.sh"

	// Test create failure
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExampleResourceEdgeCaseConfig(createScript, readScript, deleteScript),
				ExpectError: regexp.MustCompile(
					`(?s)Error: Create Script Failed.*` +
						`script execution failed with exit code 13: exit status 13.*` +
						`Exit Code: 13.*` +
						`Stdout:.*` +
						`Stderr: Failed to create resource: Permission denied.*` +
						`Input Payload: {"id":"","input":null,"output":null}`),
			},
		},
	})

	// Test delete failure
	t.Run("DeleteFailure", func(t *testing.T) {
		// Create a resource instance to test deletion
		r := &customCrudResource{}
		ctx := context.Background()

		// Set up a failing delete script
		data := customCrudResourceModel{
			Id: types.StringValue("test-123"),
			Input: types.DynamicValue(types.ObjectValueMust(
				map[string]attr.Type{
					"content": types.StringType,
				},
				map[string]attr.Value{
					"content": types.StringValue("test content"),
				},
			)),
		}

		// Create hooks block with failing delete script
		hooksObj, diags := types.ObjectValue(
			map[string]attr.Type{
				"create": types.StringType,
				"read":   types.StringType,
				"update": types.StringType,
				"delete": types.StringType,
			},
			map[string]attr.Value{
				"create": types.StringValue("../../examples/file/create.sh"),
				"read":   types.StringValue(readScript),
				"update": types.StringNull(),
				"delete": types.StringValue(deleteScript),
			},
		)
		if diags.HasError() {
			t.Fatalf("Failed to create hooks object: %v", diags)
		}

		hooksList, diags := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"create": types.StringType,
					"read":   types.StringType,
					"update": types.StringType,
					"delete": types.StringType,
				},
			},
			[]attr.Value{hooksObj},
		)
		if diags.HasError() {
			t.Fatalf("Failed to create hooks list: %v", diags)
		}

		data.Hooks = hooksList

		// Try to delete the resource
		crud, err := r.getCrudCommands(&data)
		if err != nil {
			t.Fatalf("Failed to get CRUD commands: %v", err)
		}

		deleteCmd := strings.Fields(crud.Delete.ValueString())
		result, err := r.executeScript(ctx, deleteCmd, r.convertToPayload(nil, &data))
		if err == nil {
			t.Fatal("Expected delete to fail, but it succeeded")
		}

		// Verify the error message
		errStr := fmt.Sprintf("script execution failed with exit code 7: %v\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s",
			err, result.ExitCode, result.Stdout, result.Stderr, `{"id":"test-123","input":{"content":"test content"},"output":null}`)

		if !regexp.MustCompile(
			`script execution failed with exit code 7: script execution failed with exit code 7: exit status 7\s+` +
				`Exit Code: 7\s+` +
				`Stdout:\s+` +
				`Stderr: Failed to delete resource: Resource is locked\s+` +
				`Input Payload: {"id":"test-123","input":{"content":"test content"},"output":null}`).MatchString(errStr) {
			t.Fatalf("Error message did not match expected pattern. Got: %s", errStr)
		}
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

func testAccExampleResourceEdgeCaseConfig(createScript, readScript, deleteScript string) string {
	return fmt.Sprintf(`
resource "customcrud" "test" {
  hooks {
    create = %[1]q
    read   = %[2]q
    delete = %[3]q
  }
}
`, createScript, readScript, deleteScript)
}
