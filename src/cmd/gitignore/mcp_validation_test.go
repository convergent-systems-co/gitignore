package main

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestMCPToolSchemaCompliance validates that all MCP tools comply with
// the MCP Server Standards and JSON Schema specification.
func TestMCPToolSchemaCompliance(t *testing.T) {
	tools := createMCPTools()

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			// Validate tool has a name
			if tool.Name == "" {
				t.Error("tool must have a name")
			}

			// Validate tool has a description
			if tool.Description == "" {
				t.Errorf("tool %q must have a description", tool.Name)
			}

			// Validate input schema
			validateInputSchema(t, tool.Name, tool.InputSchema)
		})
	}
}

// validateInputSchema validates that a tool's input schema complies with JSON Schema spec
func validateInputSchema(t *testing.T, toolName string, schema mcp.ToolInputSchema) {
	t.Helper()

	// Schema type must be "object" for tool parameters
	if schema.Type != "object" {
		t.Errorf("tool %q input schema type must be 'object', got %q", toolName, schema.Type)
	}

	// Validate each property in the schema
	for propName, propSchema := range schema.Properties {
		propMap, ok := propSchema.(map[string]any)
		if !ok {
			t.Errorf("tool %q property %q schema is not a valid object", toolName, propName)
			continue
		}
		validatePropertySchema(t, toolName, propName, propMap)
	}
}

// validatePropertySchema validates a single property in the tool schema
func validatePropertySchema(t *testing.T, toolName, propName string, schema map[string]any) {
	t.Helper()

	propType, hasType := schema["type"].(string)
	if !hasType {
		t.Errorf("tool %q property %q must have a 'type' field", toolName, propName)
		return
	}

	// For array types, validate items field exists (MCP Server Standard requirement)
	if propType == "array" {
		items, hasItems := schema["items"]
		if !hasItems {
			t.Errorf("tool %q property %q: array type must have 'items' field (MCP Server Standard)", toolName, propName)
			return
		}

		// Validate items schema is a valid object
		itemsMap, ok := items.(map[string]any)
		if !ok {
			t.Errorf("tool %q property %q: 'items' must be a schema object", toolName, propName)
			return
		}

		// Items must have a type
		if _, hasItemType := itemsMap["type"]; !hasItemType {
			t.Errorf("tool %q property %q: 'items' schema must have a 'type' field", toolName, propName)
		}
	}

	// For object types, validate nested properties if present
	if propType == "object" {
		if nested, hasNested := schema["properties"].(map[string]any); hasNested {
			for nestedName, nestedSchema := range nested {
				if nestedMap, ok := nestedSchema.(map[string]any); ok {
					validatePropertySchema(t, toolName, propName+"."+nestedName, nestedMap)
				}
			}
		}
	}
}

// createMCPTools creates all MCP tools for validation testing.
// This mirrors the tool definitions in cmdServe() without the handlers.
func createMCPTools() []mcp.Tool {
	return []mcp.Tool{
		// gitignore_list - no parameters
		mcp.NewTool("gitignore_list",
			mcp.WithDescription("List all available gitignore templates from configured sources (local, GitHub, Toptal)"),
		),

		// gitignore_search - string parameter
		mcp.NewTool("gitignore_search",
			mcp.WithDescription("Search for gitignore templates by name pattern"),
			mcp.WithString("pattern",
				mcp.Required(),
				mcp.Description("Search pattern to filter templates (case-insensitive substring match)"),
			),
		),

		// gitignore_add - string parameter
		mcp.NewTool("gitignore_add",
			mcp.WithDescription("Add a gitignore template to .gitignore file in the current directory"),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("Template type to add (e.g., 'go', 'github/rust', 'toptal/python')"),
			),
		),

		// gitignore_delete - string parameter
		mcp.NewTool("gitignore_delete",
			mcp.WithDescription("Remove a gitignore template section from .gitignore file"),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("Template type/section name to remove from .gitignore"),
			),
		),

		// gitignore_ignore - array parameter (must have items!)
		mcp.NewTool("gitignore_ignore",
			mcp.WithDescription("Add one or more patterns directly to .gitignore file"),
			mcp.WithArray("patterns",
				mcp.WithStringItems(),
				mcp.Required(),
				mcp.Description("Array of patterns to add to .gitignore (e.g., ['node_modules', '*.log', 'dist/'])"),
			),
		),

		// gitignore_remove - array parameter (must have items!)
		mcp.NewTool("gitignore_remove",
			mcp.WithDescription("Remove one or more patterns from .gitignore file"),
			mcp.WithArray("patterns",
				mcp.WithStringItems(),
				mcp.Required(),
				mcp.Description("Array of patterns to remove from .gitignore"),
			),
		),

		// gitignore_init - no parameters
		mcp.NewTool("gitignore_init",
			mcp.WithDescription("Initialize .gitignore with configured default template types"),
		),
	}
}

// TestMCPArrayItemsRequired specifically tests that array parameters have items defined.
// This is the specific validation that VS Code Copilot requires.
func TestMCPArrayItemsRequired(t *testing.T) {
	tools := createMCPTools()

	arrayTools := map[string]string{
		"gitignore_ignore": "patterns",
		"gitignore_remove": "patterns",
	}

	for _, tool := range tools {
		expectedArrayProp, hasArrayProp := arrayTools[tool.Name]
		if !hasArrayProp {
			continue
		}

		t.Run(tool.Name, func(t *testing.T) {
			propSchemaRaw, exists := tool.InputSchema.Properties[expectedArrayProp]
			if !exists {
				t.Fatalf("expected property %q not found in tool %q", expectedArrayProp, tool.Name)
			}

			propSchema, ok := propSchemaRaw.(map[string]any)
			if !ok {
				t.Fatalf("property %q schema is not a valid object", expectedArrayProp)
			}

			propType, _ := propSchema["type"].(string)
			if propType != "array" {
				t.Fatalf("property %q should be array type, got %q", expectedArrayProp, propType)
			}

			items, hasItems := propSchema["items"]
			if !hasItems {
				t.Errorf("property %q is missing 'items' field - this will cause MCP client validation errors", expectedArrayProp)
				return
			}

			itemsMap, ok := items.(map[string]any)
			if !ok {
				t.Errorf("property %q 'items' is not a valid schema object", expectedArrayProp)
				return
			}

			itemType, hasType := itemsMap["type"].(string)
			if !hasType {
				t.Errorf("property %q 'items' schema missing 'type' field", expectedArrayProp)
				return
			}

			if itemType != "string" {
				t.Errorf("property %q 'items' type should be 'string', got %q", expectedArrayProp, itemType)
			}
		})
	}
}

// TestMCPRequiredParameters validates that required parameters are properly marked
func TestMCPRequiredParameters(t *testing.T) {
	tools := createMCPTools()

	expectedRequired := map[string][]string{
		"gitignore_list":   {},
		"gitignore_search": {"pattern"},
		"gitignore_add":    {"type"},
		"gitignore_delete": {"type"},
		"gitignore_ignore": {"patterns"},
		"gitignore_remove": {"patterns"},
		"gitignore_init":   {},
	}

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			expected, exists := expectedRequired[tool.Name]
			if !exists {
				t.Fatalf("no expected required params defined for %q", tool.Name)
			}

			// Check that all expected required params are in the schema
			for _, param := range expected {
				found := false
				for _, req := range tool.InputSchema.Required {
					if req == param {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("parameter %q should be required", param)
				}
			}

			// Check that schema doesn't have extra required params
			for _, req := range tool.InputSchema.Required {
				found := false
				for _, exp := range expected {
					if req == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("parameter %q is marked required but wasn't expected", req)
				}
			}
		})
	}
}

// TestMCPToolCount validates that all expected tools are defined
func TestMCPToolCount(t *testing.T) {
	tools := createMCPTools()

	expectedTools := []string{
		"gitignore_list",
		"gitignore_search",
		"gitignore_add",
		"gitignore_delete",
		"gitignore_ignore",
		"gitignore_remove",
		"gitignore_init",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("expected tool %q not found", expected)
		}
	}
}
