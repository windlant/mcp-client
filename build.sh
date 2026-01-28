#!/bin/bash

go build -o ./cmd/mcp_server_local/mcp-server-local ./cmd/mcp_server_local
go build -o mcp-client ./cmd/mcp_client