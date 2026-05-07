// Copyright 2025 bluefunda
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerResources registers all MCP resources
func registerResources(server *mcp.Server, handlers *Handlers) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://program/{name}",
		Name:        "ABAP Program",
		Description: "Retrieve ABAP program source code by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleProgramResource)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://class/{name}",
		Name:        "ABAP Class",
		Description: "Retrieve ABAP class source code by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleClassResource)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://function/{group}/{name}",
		Name:        "ABAP Function Module",
		Description: "Retrieve ABAP function module source code by function group and name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleFunctionResource)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://interface/{name}",
		Name:        "ABAP Interface",
		Description: "Retrieve ABAP interface source code by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleInterfaceResource)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://table/{name}",
		Name:        "ABAP Table",
		Description: "Retrieve ABAP table structure by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleTableResource)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://structure/{name}",
		Name:        "ABAP Structure",
		Description: "Retrieve ABAP structure definition by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleStructureResource)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abap://include/{name}",
		Name:        "ABAP Include",
		Description: "Retrieve ABAP include source code by name",
		MIMEType:    "text/x-abap",
	}, handlers.HandleIncludeResource)

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

	result, err := h.apiClient.GetObject("PROG", name, "")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "PROGRAM", "", result.Source),
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

	result, err := h.apiClient.GetObject("CLAS", name, "")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "CLASS", "", result.Source),
			},
		},
	}, nil
}

// HandleFunctionResource handles function resource requests
func (h *Handlers) HandleFunctionResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// URI: abap://function/{group}/{name}
	path := strings.TrimPrefix(req.Params.URI, "abap://function/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("function resource requires format: abap://function/{group}/{name}")
	}

	group := parts[0]
	name := parts[1]

	result, err := h.apiClient.GetObject("FUNC", name, group)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "FUNCTION", "", result.Source),
			},
		},
	}, nil
}

// HandleInterfaceResource handles interface resource requests
func (h *Handlers) HandleInterfaceResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	name := extractNameFromURI(req.Params.URI, "abap://interface/")
	if name == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	result, err := h.apiClient.GetObject("INTF", name, "")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "INTERFACE", "", result.Source),
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

	result, err := h.apiClient.GetObject("TABL", name, "")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "TABLE", "", result.Source),
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

	result, err := h.apiClient.GetObject("STRU", name, "")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "STRUCTURE", "", result.Source),
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

	result, err := h.apiClient.GetObject("INCL", name, "")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/x-abap",
				Text:     formatSourceCode(name, "INCLUDE", "", result.Source),
			},
		},
	}, nil
}

// HandlePackagesResource handles packages list resource
func (h *Handlers) HandlePackagesResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	packages, err := h.apiClient.ListPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	var content strings.Builder
	content.WriteString("# ABAP Packages\n\n")
	for _, pkg := range packages {
		fmt.Fprintf(&content, "## %s\n", pkg.Name)
		if pkg.Description != "" {
			fmt.Fprintf(&content, "%s\n", pkg.Description)
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
	fmt.Fprintf(&sb, "* ABAP %s: %s\n", objType, name)
	if description != "" {
		fmt.Fprintf(&sb, "* Description: %s\n", description)
	}
	sb.WriteString("*" + strings.Repeat("-", 70) + "\n\n")
	sb.WriteString(sourceCode)
	return sb.String()
}
