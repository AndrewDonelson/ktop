package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/nlaak/ktop/internal/config"
	"github.com/nlaak/ktop/internal/models"
)

// Client wraps Kubernetes clients and provides cluster operations
type Client struct {
	config        *rest.Config
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
	clusterInfo   models.ClusterInfo
	rawConfig     *api.Config
}

// NewClient creates a new Kubernetes client from the given configuration
func NewClient(cfg *config.Config) (*Client, error) {
	var restConfig *rest.Config
	var rawConfig *api.Config
	var err error

	// Try in-cluster config first
	restConfig, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig file
		restConfig, rawConfig, err = loadKubeconfig(cfg.KubeconfigPath, cfg.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}

	// Set timeout from config
	restConfig.Timeout = cfg.Timeout

	// Create the main Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create the metrics clientset
	metricsClient, err := metricsv.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %w", err)
	}

	// Extract cluster info
	clusterInfo := extractClusterInfo(restConfig, rawConfig, cfg.Context)

	return &Client{
		config:        restConfig,
		clientset:     clientset,
		metricsClient: metricsClient,
		clusterInfo:   clusterInfo,
		rawConfig:     rawConfig,
	}, nil
}

// loadKubeconfig loads kubeconfig from file
func loadKubeconfig(kubeconfigPath, contextName string) (*rest.Config, *api.Config, error) {
	// Expand ~ in path
	if kubeconfigPath[:2] == "~/" {
		home, _ := os.UserHomeDir()
		kubeconfigPath = filepath.Join(home, kubeconfigPath[2:])
	}

	// Build config loading rules
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: kubeconfigPath,
	}

	// Build config overrides
	configOverrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		configOverrides.CurrentContext = contextName
	}

	// Create client config
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)

	// Get the raw config for extracting cluster info
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get raw kubeconfig: %w", err)
	}

	// Get the REST config
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create REST config: %w", err)
	}

	return restConfig, &rawConfig, nil
}

// extractClusterInfo extracts cluster information from configs
func extractClusterInfo(restConfig *rest.Config, rawConfig *api.Config, contextOverride string) models.ClusterInfo {
	info := models.ClusterInfo{
		Server: restConfig.Host,
	}

	if rawConfig != nil {
		// Determine current context
		contextName := rawConfig.CurrentContext
		if contextOverride != "" {
			contextName = contextOverride
		}
		info.Context = contextName

		// Get cluster name from context
		if ctx, ok := rawConfig.Contexts[contextName]; ok {
			info.Name = ctx.Cluster
			info.Namespace = ctx.Namespace
		}
	}

	// If no cluster name, use server host
	if info.Name == "" {
		info.Name = restConfig.Host
	}

	return info
}

// Clientset returns the Kubernetes clientset
func (c *Client) Clientset() *kubernetes.Clientset {
	return c.clientset
}

// MetricsClient returns the metrics clientset
func (c *Client) MetricsClient() *metricsv.Clientset {
	return c.metricsClient
}

// ClusterInfo returns information about the connected cluster
func (c *Client) ClusterInfo() models.ClusterInfo {
	return c.clusterInfo
}

// CheckMetricsAPIAvailable checks if the metrics API is available
func (c *Client) CheckMetricsAPIAvailable(ctx context.Context) error {
	_, err := c.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, 
		metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("metrics API not available: %w", err)
	}
	return nil
}

// GetContexts returns available contexts from kubeconfig
func (c *Client) GetContexts() []string {
	if c.rawConfig == nil {
		return nil
	}
	
	contexts := make([]string, 0, len(c.rawConfig.Contexts))
	for name := range c.rawConfig.Contexts {
		contexts = append(contexts, name)
	}
	return contexts
}
