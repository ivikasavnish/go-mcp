package main

import (
	"log"
	"os"

	"github.com/ivikasavnish/go-mcp/pkg/mcp"
	"github.com/ivikasavnish/go-mcp/pkg/specprocessor"
)

func main() {
	// Create and start MCP server
	server := mcp.NewServer(nil) // Using default in-memory store
	go func() {
		if err := server.Start(":6666"); err != nil {
			log.Fatalf("Failed to start MCP server: %v", err)
		}
	}()

	// Create spec processor with custom logger
	processor := specprocessor.NewProcessor(
		"http://localhost:6666",
		specprocessor.WithLogger(log.New(os.Stdout, "[PROCESSOR] ", log.LstdFlags)),
	)

	// Process API specifications
	if err := processor.ProcessDirectory("./specs"); err != nil {
		log.Fatalf("Failed to process specifications: %v", err)
	}

	// Keep the server running
	select {}
}
