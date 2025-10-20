package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"aegis-gateway/internal/adapters/files"
	"aegis-gateway/internal/adapters/payments"
	"aegis-gateway/internal/gateway"
	"aegis-gateway/pkg/telemetry"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// init telemetry first
	err := telemetry.InitTelemetry("aegis-gateway", "./logs/aegis.log")
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	defer telemetry.Close()

	// start payments adapter on port 8081
	paymentsAdapter := payments.NewAdapter()
	go func() {
		err := paymentsAdapter.Start(":8081")
		if err != nil {
			fmt.Printf("ERROR: payments adapter failed: %v\n", err)
		}
	}()

	// start files adapter on port 8082
	filesAdapter := files.NewAdapter()
	go func() {
		err := filesAdapter.Start(":8082")
		if err != nil {
			fmt.Printf("ERROR: files adapter failed: %v\n", err)
		}
	}()

	// adapter registry
	adapterMap := map[string]string{
		"payments": "http://localhost:8081",
		"files":    "http://localhost:8082",
	}

	// create gateway
	gw, err := gateway.NewGateway("./policies", adapterMap)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}
	defer gw.Close()

	// start gateway on port 8080
	go func() {
		err := gw.Start(":8080")
		if err != nil {
			fmt.Printf("ERROR: gateway failed: %v\n", err)
		}
	}()

	fmt.Println("Aegis Gateway started successfully")
	fmt.Println("Gateway: http://localhost:8080")
	fmt.Println("Payments: http://localhost:8081")
	fmt.Println("Files: http://localhost:8082")

	// wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down gracefully...")
	return nil
}
