// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"crud": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// Get the project root directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	projectRoot := filepath.Join(cwd, "..", "..")
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		t.Fatalf("Failed to get absolute path to project root: %v", err)
	}

	// Create logs directory if it doesn't exist
	logsDir := filepath.Join(projectRoot, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("Failed to create logs directory: %v", err)
	}

	logFile := filepath.Join(logsDir, "terraform-provider-customcrud.log")
	t.Logf("Log file path: %s", logFile)

	// Enable debug logging
	t.Setenv("TF_LOG", "DEBUG")
	t.Setenv("TF_LOG_PATH", logFile)
}
