package provider

import (
	"context"
	"testing"

	"github.com/customcrud/terraform-provider-customcrud/internal/provider/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
)

// mockPrivate implements the privater interface for testing.
type mockPrivate struct {
	data map[string][]byte
}

func (m *mockPrivate) GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics) {
	return m.data[key], nil
}

func TestUnitCustomCrudEphemeral_Metadata(t *testing.T) {
	e := NewCustomCrudEphemeral()
	req := ephemeral.MetadataRequest{}
	resp := &ephemeral.MetadataResponse{}

	e.Metadata(context.Background(), req, resp)

	if resp.TypeName != "customcrud" {
		t.Errorf("Expected TypeName customcrud, got %s", resp.TypeName)
	}
}

func TestUnitCustomCrudEphemeral_Schema(t *testing.T) {
	e := NewCustomCrudEphemeral()
	req := ephemeral.SchemaRequest{}
	resp := &ephemeral.SchemaResponse{}

	e.Schema(context.Background(), req, resp)

	if _, ok := resp.Schema.Attributes["input"]; !ok {
		t.Error("Schema should have input attribute")
	}
}

func TestUnitCustomCrudEphemeral_Configure(t *testing.T) {
	e := &customCrudEphemeral{}

	// Test nil provider data
	req := ephemeral.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &ephemeral.ConfigureResponse{}
	e.Configure(context.Background(), req, resp)
	if e.config.Parallelism != 0 {
		t.Error("Expected default config on nil ProviderData")
	}

	// Test valid provider data
	p := &CustomCRUDProvider{
		config: utils.CustomCRUDProviderConfig{
			Parallelism: 5,
		},
	}
	req.ProviderData = p
	e.Configure(context.Background(), req, resp)
	if e.config.Parallelism != 5 {
		t.Errorf("Expected parallelism 5, got %d", e.config.Parallelism)
	}
}

func TestUnitCustomCrudEphemeral_Renew_NoHook(t *testing.T) {
	e := &customCrudEphemeral{}
	ctx := context.Background()

	// Mock private state with no renew hook
	private := &mockPrivate{
		data: map[string][]byte{
			"hooks": []byte(`{"open": "echo open"}`),
		},
	}

	diags := &diag.Diagnostics{}
	e.renew(ctx, private, diags)

	if diags.HasError() {
		t.Errorf("Unexpected error in Renew without hook: %v", diags)
	}
}

func TestUnitCustomCrudEphemeral_Close_NoHook(t *testing.T) {
	e := &customCrudEphemeral{}
	ctx := context.Background()

	// Mock private state with no close hook
	private := &mockPrivate{
		data: map[string][]byte{
			"hooks": []byte(`{"open": "echo open"}`),
		},
	}

	diags := &diag.Diagnostics{}
	e.close(ctx, private, diags)

	if diags.HasError() {
		t.Errorf("Unexpected error in Close without hook: %v", diags)
	}
}

func TestUnitCustomCrudEphemeral_Renew_Success(t *testing.T) {
	e := &customCrudEphemeral{}
	ctx := context.Background()

	// Mock private state with renew hook
	private := &mockPrivate{
		data: map[string][]byte{
			"hooks":  []byte(`{"open": "echo open", "renew": "true"}`),
			"input":  []byte(`{"foo": "bar"}`),
			"output": []byte(`{"status": "ok"}`),
		},
	}

	diags := &diag.Diagnostics{}
	e.renew(ctx, private, diags)

	if diags.HasError() {
		t.Errorf("Unexpected error in Renew: %v", diags)
	}
}

func TestUnitCustomCrudEphemeral_Close_Success(t *testing.T) {
	e := &customCrudEphemeral{}
	ctx := context.Background()

	// Mock private state with close hook
	private := &mockPrivate{
		data: map[string][]byte{
			"hooks":  []byte(`{"open": "echo open", "close": "true"}`),
			"input":  []byte(`{"foo": "bar"}`),
			"output": []byte(`{"status": "ok"}`),
		},
	}

	diags := &diag.Diagnostics{}
	e.close(ctx, private, diags)

	if diags.HasError() {
		t.Errorf("Unexpected error in Close: %v", diags)
	}
}

func TestUnitCustomCrudEphemeral_Renew_UnmarshalError(t *testing.T) {
	e := &customCrudEphemeral{}
	ctx := context.Background()

	// Mock private state with invalid JSON in hooks
	private := &mockPrivate{
		data: map[string][]byte{
			"hooks": []byte(`{invalid json`),
		},
	}

	diags := &diag.Diagnostics{}
	e.renew(ctx, private, diags)

	if !diags.HasError() {
		t.Error("Expected error in Renew with invalid hooks JSON")
	}
}

func TestUnitCustomCrudEphemeral_Close_UnmarshalError(t *testing.T) {
	e := &customCrudEphemeral{}
	ctx := context.Background()

	// Mock private state with invalid JSON in hooks
	private := &mockPrivate{
		data: map[string][]byte{
			"hooks": []byte(`{invalid json`),
		},
	}

	diags := &diag.Diagnostics{}
	e.close(ctx, private, diags)

	if !diags.HasError() {
		t.Error("Expected error in Close with invalid hooks JSON")
	}
}

func TestUnitCustomCrudEphemeral_PublicWrappers(t *testing.T) {
	e := &customCrudEphemeral{}
	ctx := context.Background()

	// Call Renew with default request (nil Private)
	// This will hit the entry point and then return in renew logic because priv is nil
	e.Renew(ctx, ephemeral.RenewRequest{}, &ephemeral.RenewResponse{})

	// Call Close with default request (nil Private)
	e.Close(ctx, ephemeral.CloseRequest{}, &ephemeral.CloseResponse{})
}
