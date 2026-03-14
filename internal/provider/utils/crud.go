package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"mvdan.cc/sh/v3/shell"
)

// CrudHooks is a generic struct for CRUD command strings
// (for resource: create, read, update, delete; for data source: just read;
// for ephemeral resource: open, renew, close).
type CrudHooks struct {
	Create types.String
	Read   types.String
	Update types.String
	Delete types.String
	Open   types.String
	Renew  types.String
	Close  types.String
}

// CrudModel is an interface for models that have a Hooks field (types.List).
type CrudModel interface {
	GetHooks() types.List
}

// getCrudCommands extracts CRUD commands from a model implementing CrudModel.
func GetCrudCommands(model CrudModel) (*CrudHooks, error) {
	hooks := model.GetHooks()
	if hooks.IsNull() || hooks.IsUnknown() {
		return nil, fmt.Errorf("hooks block is null or unknown")
	}
	elements := hooks.Elements()
	if len(elements) == 0 {
		return nil, fmt.Errorf("hooks block is empty")
	}
	obj, ok := elements[0].(types.Object)
	if !ok {
		return nil, fmt.Errorf("hooks block element is not an object")
	}
	attrs := obj.Attributes()
	crud := &CrudHooks{}
	if create, ok := attrs[Create].(types.String); ok {
		crud.Create = create
	}
	if read, ok := attrs[Read].(types.String); ok {
		crud.Read = read
	}
	if update, ok := attrs[Update].(types.String); ok {
		crud.Update = update
	}
	if del, ok := attrs[Delete].(types.String); ok {
		crud.Delete = del
	}
	if open, ok := attrs[Open].(types.String); ok {
		crud.Open = open
	}
	if renew, ok := attrs[Renew].(types.String); ok {
		crud.Renew = renew
	}
	if closeHook, ok := attrs[Close].(types.String); ok {
		crud.Close = closeHook
	}
	return crud, nil
}

type CrudOp int

const Create = "create"
const Read = "read"
const Update = "update"
const Delete = "delete"
const Open = "open"
const Renew = "renew"
const Close = "close"
const Unknown = "unknown"

const (
	CrudCreate CrudOp = iota
	CrudRead
	CrudUpdate
	CrudDelete
	CrudOpen
	CrudRenew
	CrudClose
)

func (op CrudOp) String() string {
	switch op {
	case CrudCreate:
		return Create
	case CrudRead:
		return Read
	case CrudUpdate:
		return Update
	case CrudDelete:
		return Delete
	case CrudOpen:
		return Open
	case CrudRenew:
		return Renew
	case CrudClose:
		return Close
	default:
		return Unknown
	}
}

type CustomCRUDProviderConfig struct {
	Parallelism          int
	HighPrecisionNumbers bool
	Semaphore            chan struct{}
	DefaultInputs        interface{}
}

func CustomCRUDProviderConfigDefaults() CustomCRUDProviderConfig {
	return CustomCRUDProviderConfig{
		Parallelism:          0,
		HighPrecisionNumbers: false,
		Semaphore:            nil,
		DefaultInputs:        nil,
	}
}

// RunCrudScript runs the appropriate CRUD script for the given op (CrudCreate, CrudRead, CrudUpdate, CrudDelete)
// and handles error/diagnostic reporting. The model must implement CrudModel.
func RunCrudScript(ctx context.Context, config CustomCRUDProviderConfig, model CrudModel, payload ExecutionPayload, diagnostics *diag.Diagnostics, op CrudOp) (*ExecutionResult, bool) {
	crud, err := GetCrudCommands(model)
	if err != nil {
		diagnostics.AddError("Error getting CRUD commands", err.Error())
		return nil, false
	}
	var commandStr string
	switch op {
	case CrudCreate:
		commandStr = crud.Create.ValueString()
	case CrudRead:
		commandStr = crud.Read.ValueString()
	case CrudUpdate:
		commandStr = crud.Update.ValueString()
	case CrudDelete:
		commandStr = crud.Delete.ValueString()
	case CrudOpen:
		commandStr = crud.Open.ValueString()
	case CrudRenew:
		commandStr = crud.Renew.ValueString()
	case CrudClose:
		commandStr = crud.Close.ValueString()
	default:
		diagnostics.AddError("Invalid Operation", fmt.Sprintf("Unknown operation: %v", op))
		return nil, false
	}
	cmd, err := shell.Fields(commandStr, nil)
	if err != nil {
		diagnostics.AddError(fmt.Sprintf("Invalid %v Command", op), fmt.Sprintf("failed to parse %v command: %v", op, err))
		return nil, false
	}
	if len(cmd) == 0 {
		diagnostics.AddError(fmt.Sprintf("Invalid %v Command", op), fmt.Sprintf("%v command cannot be empty", op))
		return nil, false
	}
	result, err := Execute(ctx, config, cmd, payload)

	title := cases.Title(language.English)
	if err != nil {
		// Special case: for Read operations with exit code 22, don't add error diagnostic
		if op == CrudRead && result != nil && result.ExitCode == 22 {
			return result, false
		}
		payloadJSON, _ := json.Marshal(payload)
		diagnostics.AddError(fmt.Sprintf("%v Script Failed", title.String(op.String())), fmt.Sprintf("%v\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s", err, result.ExitCode, result.Stdout, result.Stderr, string(payloadJSON)))
		return result, false
	}
	// For delete operations, nil output is expected and should not be treated as an error
	if result == nil || (result.Result == nil && op != CrudDelete) {
		payloadJSON, _ := json.Marshal(payload)
		diagnostics.AddError(fmt.Sprintf("%v Script Failed", title.String(op.String())), fmt.Sprintf("%v script returned nil output\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s", op, result.ExitCode, result.Stdout, result.Stderr, string(payloadJSON)))
		return result, false
	}
	return result, true
}
