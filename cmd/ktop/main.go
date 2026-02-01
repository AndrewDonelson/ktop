package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nlaak/ktop/internal/config"
	"github.com/nlaak/ktop/internal/k8s"
	"github.com/nlaak/ktop/internal/metrics"
	"github.com/nlaak/ktop/internal/ui"
)

func main() {
	// Parse configuration
	cfg := config.NewConfig()
	if err := cfg.ParseFlags(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Handle version flag
	if cfg.ShowVersion {
		config.PrintVersion()
		os.Exit(0)
	}

	// Handle help flag (already handled by flag package, but just in case)
	if cfg.ShowHelp {
		os.Exit(0)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Initialize Kubernetes client
	fmt.Println("Connecting to Kubernetes cluster...")
	client, err := k8s.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to cluster: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nMake sure:\n")
		fmt.Fprintf(os.Stderr, "  - Your kubeconfig is valid (default: ~/.kube/config)\n")
		fmt.Fprintf(os.Stderr, "  - The cluster is accessible\n")
		fmt.Fprintf(os.Stderr, "  - You have permission to access cluster resources\n")
		os.Exit(1)
	}

	// Check if metrics API is available
	checkCtx, checkCancel := context.WithTimeout(ctx, 5*time.Second)
	defer checkCancel()
	
	if err := client.CheckMetricsAPIAvailable(checkCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Metrics API not available: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please ensure metrics-server is installed in your cluster.\n")
		fmt.Fprintf(os.Stderr, "\nTo install metrics-server:\n")
		fmt.Fprintf(os.Stderr, "  kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml\n")
		fmt.Fprintf(os.Stderr, "\nFor MicroK8s:\n")
		fmt.Fprintf(os.Stderr, "  microk8s enable metrics-server\n")
		fmt.Fprintf(os.Stderr, "\nContinuing anyway (some features may not work)...\n")
		time.Sleep(2 * time.Second)
	}

	// Print cluster info
	info := client.ClusterInfo()
	fmt.Printf("Connected to cluster: %s (context: %s)\n", info.Name, info.Context)
	fmt.Println("Starting ktop...")

	// Create metrics collector
	collector := metrics.NewCollector(client, cfg)

	// Create and run the TUI application
	app := ui.NewApp(collector, cfg)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}
