package main

import "github.com/mark3labs/mcp-go/mcp"

// mcpToolDefs returns the static schemas for every MCP tool the gitignore
// server exposes. Handlers are bound separately in cmdServe so this function
// has no dependency on runtime config and can be exercised by schema tests.
func mcpToolDefs() []mcp.Tool {
	return []mcp.Tool{
		mcp.NewTool("gitignore_list",
			mcp.WithDescription("List all available gitignore templates from configured sources (local, GitHub, Toptal)"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),

		mcp.NewTool("gitignore_search",
			mcp.WithDescription("Search for gitignore templates by name pattern"),
			mcp.WithString("pattern",
				mcp.Required(),
				mcp.Description("Search pattern to filter templates (case-insensitive substring match)"),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),

		mcp.NewTool("gitignore_add",
			mcp.WithDescription("Add a gitignore template to .gitignore file in the current directory"),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("Template type to add (e.g., 'go', 'github/rust', 'toptal/python')"),
			),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),

		mcp.NewTool("gitignore_delete",
			mcp.WithDescription("Remove a gitignore template section from .gitignore file"),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("Template type/section name to remove from .gitignore"),
			),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),

		mcp.NewTool("gitignore_ignore",
			mcp.WithDescription("Add one or more patterns directly to .gitignore file"),
			mcp.WithArray("patterns",
				mcp.WithStringItems(),
				mcp.Required(),
				mcp.Description("Array of patterns to add to .gitignore (e.g., ['node_modules', '*.log', 'dist/'])"),
			),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),

		mcp.NewTool("gitignore_remove",
			mcp.WithDescription("Remove one or more patterns from .gitignore file"),
			mcp.WithArray("patterns",
				mcp.WithStringItems(),
				mcp.Required(),
				mcp.Description("Array of patterns to remove from .gitignore"),
			),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),

		mcp.NewTool("gitignore_init",
			mcp.WithDescription("Initialize .gitignore with configured default template types"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
	}
}
