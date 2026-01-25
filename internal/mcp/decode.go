package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// decode unmarshals MCP request arguments into a typed struct.
// Avoids unsafe type assertions and handles JSON decoding safely.
func decode[T any](req mcp.CallToolRequest) (T, error) {
	var result T
	args := req.GetArguments()
	b, err := json.Marshal(args)
	if err != nil {
		return result, fmt.Errorf("marshal args: %w", err)
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return result, fmt.Errorf("unmarshal args: %w", err)
	}
	return result, nil
}
