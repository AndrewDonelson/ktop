package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Version is set at build time via ldflags
// Format: YYYYMMDD.HHMM-{alpha|beta|prod}
var Version = "dev"

const (
	// DefaultRefreshInterval is the default metrics refresh interval
	DefaultRefreshInterval = 2 * time.Second

	// DefaultTimeout is the default API call timeout
	DefaultTimeout = 10 * time.Second

	// DefaultTopPods is the default number of pods to display
	DefaultTopPods = 30

	// MinRefreshInterval is the minimum allowed refresh interval
	MinRefreshInterval = 500 * time.Millisecond

	// MaxRefreshInterval is the maximum allowed refresh interval
	MaxRefreshInterval = 60 * time.Second
)

// Config holds all configuration options for ktop
type Config struct {
	// Kubernetes configuration
	KubeconfigPath string
	Context        string

	// Display configuration
	RefreshInterval time.Duration
	Timeout         time.Duration
	TopPods         int
	AllNamespaces   bool

	// Flags
	ShowVersion bool
	ShowHelp    bool
}

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		KubeconfigPath:  defaultKubeconfigPath(),
		Context:         "",
		RefreshInterval: DefaultRefreshInterval,
		Timeout:         DefaultTimeout,
		TopPods:         DefaultTopPods,
		AllNamespaces:   false,
		ShowVersion:     false,
		ShowHelp:        false,
	}
}

// defaultKubeconfigPath returns the default kubeconfig path
func defaultKubeconfigPath() string {
	// Check KUBECONFIG environment variable first
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	// Fall back to ~/.kube/config
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

// ParseFlags parses command-line flags and populates the config
func (c *Config) ParseFlags() error {
	flag.StringVar(&c.KubeconfigPath, "kubeconfig", c.KubeconfigPath,
		"Path to kubeconfig file")
	flag.StringVar(&c.Context, "context", c.Context,
		"Kubernetes context to use (default: current context)")
	flag.DurationVar(&c.RefreshInterval, "refresh-interval", c.RefreshInterval,
		"Metrics refresh interval (e.g., 2s, 5s)")
	flag.DurationVar(&c.Timeout, "timeout", c.Timeout,
		"API call timeout (e.g., 10s, 30s)")
	flag.IntVar(&c.TopPods, "top-pods", c.TopPods,
		"Number of top pods to display")
	flag.BoolVar(&c.AllNamespaces, "all-namespaces", c.AllNamespaces,
		"Include system namespaces (kube-system, etc.)")
	flag.BoolVar(&c.ShowVersion, "version", c.ShowVersion,
		"Show version information")
	flag.BoolVar(&c.ShowHelp, "help", c.ShowHelp,
		"Show help message")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ktop - Kubernetes Cluster Monitor (v%s)\n\n", Version)
		fmt.Fprintf(os.Stderr, "Usage: ktop [options]\n\n")
		fmt.Fprintf(os.Stderr, "A terminal UI for monitoring Kubernetes cluster resources,\n")
		fmt.Fprintf(os.Stderr, "similar to htop for Linux processes.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nKeyboard Controls:\n")
		fmt.Fprintf(os.Stderr, "  q          Quit\n")
		fmt.Fprintf(os.Stderr, "  r          Force refresh\n")
		fmt.Fprintf(os.Stderr, "  s          Sort nodes (cycle: name, CPU, memory, status)\n")
		fmt.Fprintf(os.Stderr, "  p          Sort pods (cycle: namespace, name, CPU, memory)\n")
		fmt.Fprintf(os.Stderr, "  f          Filter pods by namespace\n")
		fmt.Fprintf(os.Stderr, "  n          Next namespace filter\n")
		fmt.Fprintf(os.Stderr, "  t          Toggle view mode (split/nodes/pods)\n")
		fmt.Fprintf(os.Stderr, "  a          Toggle system namespaces\n")
		fmt.Fprintf(os.Stderr, "  ?          Show help\n")
		fmt.Fprintf(os.Stderr, "  ↑/↓        Navigate selection\n")
		fmt.Fprintf(os.Stderr, "  Tab        Switch between nodes and pods\n")
		fmt.Fprintf(os.Stderr, "\nRequirements:\n")
		fmt.Fprintf(os.Stderr, "  - Kubernetes cluster with metrics-server installed\n")
		fmt.Fprintf(os.Stderr, "  - Valid kubeconfig file\n")
	}

	flag.Parse()

	return c.Validate()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.RefreshInterval < MinRefreshInterval {
		return fmt.Errorf("refresh interval must be at least %v", MinRefreshInterval)
	}
	if c.RefreshInterval > MaxRefreshInterval {
		return fmt.Errorf("refresh interval must not exceed %v", MaxRefreshInterval)
	}
	if c.Timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second")
	}
	if c.TopPods < 1 {
		return fmt.Errorf("top-pods must be at least 1")
	}
	if c.TopPods > 1000 {
		return fmt.Errorf("top-pods should not exceed 1000")
	}
	return nil
}

// PrintVersion prints version information
func PrintVersion() {
	fmt.Printf("ktop %s\n", Version)
	fmt.Println("Kubernetes Cluster Monitor (Official)")
	fmt.Println()
	fmt.Println("Copyright (c) 2026 Nlaak Studios")
	fmt.Println("Author:  Andrew Donelson <https://andrewdonelson.com>")
	fmt.Println("Website: https://nlaak.com")
	fmt.Println("Source:  https://github.com/andrewdonelson/ktop")
}

// ThresholdConfig holds threshold values for color coding
type ThresholdConfig struct {
	WarningPercent  float64 // threshold for yellow (default 50%)
	CriticalPercent float64 // threshold for red (default 80%)
}

// DefaultThresholds returns default threshold configuration
func DefaultThresholds() ThresholdConfig {
	return ThresholdConfig{
		WarningPercent:  50.0,
		CriticalPercent: 80.0,
	}
}
