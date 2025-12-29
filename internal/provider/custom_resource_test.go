// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/customcrud/terraform-provider-customcrud/internal/provider/utils"
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
				ImportStateIdFunc:       testAccResourceImportStateIdFunc("customcrud.test", "", createScript, readScript, updateScript, deleteScript),
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

					resource.TestCheckResourceAttr("customcrud.test", "output.b.c.#", "3"),
					resource.TestCheckResourceAttr("customcrud.test", "output.b.c.0", "a"),
					resource.TestCheckResourceAttr("customcrud.test", "output.b.c.1", "b"),
					resource.TestCheckResourceAttr("customcrud.test", "output.b.c.2", "c"),
					resource.TestCheckResourceAttr("customcrud.test", "output.b.d.#", "0"),
				),
				// jq -n '{id: 1, a: [1, "2", false, null, [{"b": 3}], [1, 2, 3]]}'
			},
			// Import testing
			{
				Config:                  testAccExampleResourceEdgeCaseConfig(createScript, readScript, deleteScript),
				ResourceName:            "customcrud.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccResourceImportStateIdFunc("customcrud.test", "{\"input\":{\"b\":{\"c\":[\"a\",\"b\",\"c\"],\"d\":[]}}}", createScript, readScript, "", deleteScript),
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
						`Input Payload: .*`),
			},
		},
	})

	// Test delete failure
	t.Run("DeleteFailure", func(t *testing.T) {
		// Create a resource instance to test deletion
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
				utils.Create: types.StringType,
				utils.Read:   types.StringType,
				utils.Update: types.StringType,
				utils.Delete: types.StringType,
			},
			map[string]attr.Value{
				utils.Create: types.StringValue("../../examples/file/create.sh"),
				utils.Read:   types.StringValue(readScript),
				utils.Update: types.StringNull(),
				utils.Delete: types.StringValue(deleteScript),
			},
		)
		if diags.HasError() {
			t.Fatalf("Failed to create hooks object: %v", diags)
		}

		hooksList, diags := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					utils.Create: types.StringType,
					utils.Read:   types.StringType,
					utils.Update: types.StringType,
					utils.Delete: types.StringType,
				},
			},
			[]attr.Value{hooksObj},
		)
		if diags.HasError() {
			t.Fatalf("Failed to create hooks list: %v", diags)
		}

		data.Hooks = hooksList

		// Try to delete the resource
		crud, err := getCrudCommands(&data)
		if err != nil {
			t.Fatalf("Failed to get CRUD commands: %v", err)
		}

		deleteCmd := strings.Fields(crud.Delete.ValueString())
		result, err := utils.Execute(ctx, deleteCmd, utils.ExecutionPayload{
			Id:     data.Id.ValueString(),
			Input:  utils.AttrValueToInterface(data.Input.UnderlyingValue()),
			Output: nil,
		})
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

func TestAccResourceRemovedRemote(t *testing.T) {
	createScript := "../../examples/file/hooks/create.sh"
	readScript := "../../examples/file/hooks/read.sh"
	deleteScript := "../../examples/file/hooks/delete.sh"
	readScriptSimulateRemoval := "test_resource_removed_remote/read_simulate_removed_remote.sh"

	content := "Test content for remote removal"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Simulate the resource being removed from state, as when doing a refresh, I should get a non-empty plan
			{
				Config: testAccResourceRemovedRemoteConfig(createScript, readScriptSimulateRemoval, deleteScript, content),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("customcrud.test", "output.content", content),
					resource.TestCheckResourceAttrSet("customcrud.test", "id"),
				),
				ExpectNonEmptyPlan: true,
			},
			// Then use normal read script, to verify creation
			{
				Config: testAccResourceRemovedRemoteConfig(createScript, readScript, deleteScript, content),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("customcrud.test", "output.content", content),
					resource.TestCheckResourceAttrSet("customcrud.test", "id"),
				),
			},
		},
	})
}

func TestAccParallelism_SerializesExecution(t *testing.T) {

	dir, err := filepath.Abs("test_parallel")
	if err != nil {
		t.Fatalf("Failed to resolve test_parallel dir: %v", err)
	}
	createScript := filepath.Join(dir, "create.sh")
	readScript := filepath.Join(dir, "read.sh")
	deleteScript := filepath.Join(dir, "delete.sh")

	t.Run("parallelism=1 passes", func(t *testing.T) {
		config := fmt.Sprintf(`
provider "customcrud" {
  parallelism = 1
}
resource "customcrud" "locktest_serial" {
  count = 2
  hooks {
    create = %q
    read   = %q
    delete = %q
  }
  input = { name = "lock_parallel_1" }
}
`, createScript, readScript, deleteScript)
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("customcrud.locktest_serial.0", "id"),
						resource.TestCheckResourceAttrSet("customcrud.locktest_serial.1", "id"),
					),
				},
			},
		})
	})

	t.Run("parallelism=2 fails", func(t *testing.T) {
		config := fmt.Sprintf(`
	
		provider "customcrud" {
		  parallelism = 2
		}
	
		resource "customcrud" "locktest_parallel" {
		  count = 2
		  hooks {
		    create = %q
		    read   = %q
		    delete = %q
		  }
		  input = { name = "lock_parallel_2" }
		}
	
	`, createScript, readScript, deleteScript)

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile(`(?s)Create Script Failed.*lock \[lock_parallel_2\] already held`),
				},
			},
		})
	})
}

// Helper function to generate import state ID.
func testAccResourceImportStateIdFunc(resourceName, importString string, createScript, readScript, updateScript, deleteScript string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		importData := importStateData{
			Hooks: map[string]string{
				utils.Create: createScript,
				utils.Read:   readScript,
				utils.Update: updateScript,
				utils.Delete: deleteScript,
			},
		}

		importData.Id = rs.Primary.ID
		// allow additional input/output to be tested using import string
		if importString != "" {
			var parsedImport map[string]interface{}
			if err := json.Unmarshal([]byte(importString), &parsedImport); err != nil {
				return "", fmt.Errorf("failed to parse import string as JSON: %v", err)
			}
			if input, ok := parsedImport["input"]; ok {
				if inputMap, ok := input.(map[string]interface{}); ok {
					importData.Input = inputMap
				}
			}
			if output, ok := parsedImport["output"]; ok {
				if outputMap, ok := output.(map[string]interface{}); ok {
					importData.Output = outputMap
				}
			}
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

//nolint:unparam
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

  input = {
    b = {
      c = ["a", "b", "c"]
	  d = []
    }
  }
}
`, createScript, readScript, deleteScript)
}

func testAccResourceRemovedRemoteConfig(createScript, readScript, deleteScript, content string) string {
	return fmt.Sprintf(`
resource "customcrud" "test" {
  hooks {
    create = %[1]q
    read   = %[2]q
    delete = %[3]q
  }
  input = {
    content = %[4]q
  }
}
`, createScript, readScript, deleteScript, content)
}
