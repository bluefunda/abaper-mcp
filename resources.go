package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerResources registers all MCP resources
func registerResources(server *mcp.Server, handlers *Handlers) {
	// Resource template for ABAP programs
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://program/{name}",
		Name:        "ABAP Program",
		Description: "Retrieve ABAP program source code by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleProgramResource)

	// Resource template for ABAP classes
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://class/{name}",
		Name:        "ABAP Class",
		Description: "Retrieve ABAP class source code by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleClassResource)

	// Resource template for ABAP functions
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://function/{name}",
		Name:        "ABAP Function",
		Description: "Retrieve ABAP function module source code by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleFunctionResource)

	// Resource template for ABAP interfaces
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://interface/{name}",
		Name:        "ABAP Interface",
		Description: "Retrieve ABAP interface source code by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleInterfaceResource)

	// Resource template for ABAP tables
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://table/{name}",
		Name:        "ABAP Table",
		Description: "Retrieve ABAP table structure by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleTableResource)

	// Resource template for ABAP structures
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://structure/{name}",
		Name:        "ABAP Structure",
		Description: "Retrieve ABAP structure definition by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleStructureResource)

	// Resource template for ABAP includes
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://include/{name}",
		Name:        "ABAP Include",
		Description: "Retrieve ABAP include source code by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleIncludeResource)

	// Resource for listing packages
	server.AddResource(&mcp.Resource{
		URI:         "abap://packages",
		Name:        "ABAP Packages",
		Description: "List all ABAP packages in the system",
		MIMEType:    "application/json",
	}, handlers.HandlePackagesResource)
}

// HandleProgramResource handles program resource requests
func (h *Handlers) HandleProgramResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	name := extractNameFromURI(req.Params.URI, "abap://program/")
	if name == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	client, err := h.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ADT client: %w", err)
	}

	source, err := client.GetProgram(name)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "PROGRAM", "", source.Source),
			},
		},
	}, nil
}

// HandleClassResource handles class resource requests
func (h *Handlers) HandleClassResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	name := extractNameFromURI(req.Params.URI, "abap://class/")
	if name == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	client, err := h.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ADT client: %w", err)
	}

	source, err := client.GetClass(name)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "CLASS", "", source.Source),
			},
		},
	}, nil
}

// HandleFunctionResource handles function resource requests
func (h *Handlers) HandleFunctionResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	name := extractNameFromURI(req.Params.URI, "abap://function/")
	if name == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	// Function modules require function group parameter
	// For now, return an error - this can be enhanced later
	return nil, fmt.Errorf("function modules require function_group parameter - use abap://function/{group}/{name} format or use get-object tool")
}

// HandleInterfaceResource handles interface resource requests
func (h *Handlers) HandleInterfaceResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	name := extractNameFromURI(req.Params.URI, "abap://interface/")
	if name == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	client, err := h.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ADT client: %w", err)
	}

	source, err := client.GetInterface(name)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "INTERFACE", "", source.Source),
			},
		},
	}, nil
}

// HandleTableResource handles table resource requests
func (h *Handlers) HandleTableResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	name := extractNameFromURI(req.Params.URI, "abap://table/")
	if name == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	client, err := h.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ADT client: %w", err)
	}

	source, err := client.GetTable(name)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "TABLE", "", source.Source),
			},
		},
	}, nil
}

// HandleStructureResource handles structure resource requests
func (h *Handlers) HandleStructureResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	name := extractNameFromURI(req.Params.URI, "abap://structure/")
	if name == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	client, err := h.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ADT client: %w", err)
	}

	source, err := client.GetStructure(name)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "STRUCTURE", "", source.Source),
			},
		},
	}, nil
}

// HandleIncludeResource handles include resource requests
func (h *Handlers) HandleIncludeResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	name := extractNameFromURI(req.Params.URI, "abap://include/")
	if name == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	client, err := h.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ADT client: %w", err)
	}

	source, err := client.GetInclude(name)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "INCLUDE", "", source.Source),
			},
		},
	}, nil
}

// HandlePackagesResource handles packages list resource
func (h *Handlers) HandlePackagesResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	client, err := h.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ADT client: %w", err)
	}

	packages, err := client.ListPackages("*")
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	var content strings.Builder
	content.WriteString("# ABAP Packages\n\n")
	for _, pkg := range packages {
		content.WriteString(fmt.Sprintf("## %s\n", pkg.Name))
		if pkg.Description != "" {
			content.WriteString(fmt.Sprintf("%s\n", pkg.Description))
		}
		content.WriteString("\n")
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/markdown",
				Text:     content.String(),
			},
		},
	}, nil
}

// extractNameFromURI extracts the object name from a URI
func extractNameFromURI(uri, prefix string) string {
	if !strings.HasPrefix(uri, prefix) {
		return ""
	}
	return strings.TrimPrefix(uri, prefix)
}

// formatSourceCode formats source code with metadata
func formatSourceCode(name, objType, description, sourceCode string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("* ABAP %s: %s\n", objType, name))
	if description != "" {
		sb.WriteString(fmt.Sprintf("* Description: %s\n", description))
	}
	sb.WriteString("*" + strings.Repeat("-", 70) + "\n\n")
	sb.WriteString(sourceCode)
	return sb.String()
}
