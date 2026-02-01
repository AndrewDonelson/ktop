package metrics

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nlaak/ktop/internal/config"
	"github.com/nlaak/ktop/internal/k8s"
	"github.com/nlaak/ktop/internal/models"
)

// Collector fetches and aggregates Kubernetes metrics
type Collector struct {
	client *k8s.Client
	config *config.Config

	// Cached data
	mu          sync.RWMutex
	lastMetrics *models.ClusterMetrics
	namespaces  []string
}

// NewCollector creates a new metrics collector
func NewCollector(client *k8s.Client, cfg *config.Config) *Collector {
	return &Collector{
		client: client,
		config: cfg,
	}
}

// Collect fetches all metrics from the cluster
func (c *Collector) Collect(ctx context.Context) (*models.ClusterMetrics, error) {
	metrics := &models.ClusterMetrics{
		Timestamp:   time.Now(),
		ClusterInfo: c.client.ClusterInfo(),
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Fetch node data (status, capacity, etc.)
	nodes, err := c.fetchNodes(ctx)
	if err != nil {
		metrics.Error = fmt.Errorf("failed to fetch nodes: %w", err)
		return metrics, err
	}

	// Fetch node metrics
	nodeMetrics, err := c.fetchNodeMetrics(ctx)
	if err != nil {
		// Metrics might not be available, continue with what we have
		metrics.Error = fmt.Errorf("failed to fetch node metrics: %w", err)
	}

	// Merge node info with metrics
	metrics.Nodes = c.mergeNodeData(nodes, nodeMetrics)

	// Fetch pod data and metrics
	pods, err := c.fetchPodMetrics(ctx)
	if err != nil {
		if metrics.Error == nil {
			metrics.Error = fmt.Errorf("failed to fetch pod metrics: %w", err)
		}
	}
	metrics.Pods = pods

	// Calculate aggregates
	c.calculateAggregates(metrics)

	// Update cache
	c.mu.Lock()
	c.lastMetrics = metrics
	c.mu.Unlock()

	return metrics, nil
}

// fetchNodes fetches node information from the API
func (c *Collector) fetchNodes(ctx context.Context) ([]corev1.Node, error) {
	nodeList, err := c.client.Clientset().CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

// nodeMetricData holds raw node metrics from the metrics API
type nodeMetricData struct {
	CPU    int64 // millicores
	Memory int64 // bytes
}

// fetchNodeMetrics fetches metrics from the metrics API
func (c *Collector) fetchNodeMetrics(ctx context.Context) (map[string]nodeMetricData, error) {
	metricsList, err := c.client.MetricsClient().MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make(map[string]nodeMetricData)
	for _, nm := range metricsList.Items {
		result[nm.Name] = nodeMetricData{
			CPU:    nm.Usage.Cpu().MilliValue(),
			Memory: nm.Usage.Memory().Value(),
		}
	}
	return result, nil
}

// mergeNodeData combines node status with metrics
func (c *Collector) mergeNodeData(nodes []corev1.Node, metrics map[string]nodeMetricData) []models.Node {
	result := make([]models.Node, 0, len(nodes))

	for _, n := range nodes {
		node := models.Node{
			Name:   n.Name,
			Labels: n.Labels,
		}

		// Get node status
		node.Status = c.getNodeStatus(n)

		// Get capacity
		cpuCap := n.Status.Capacity.Cpu()
		memCap := n.Status.Capacity.Memory()

		node.CPU.Capacity = cpuCap.MilliValue()
		node.Memory.Capacity = memCap.Value()

		// Get disk capacity if available
		if ephStorage, ok := n.Status.Capacity[corev1.ResourceEphemeralStorage]; ok {
			node.Disk.Capacity = ephStorage.Value()
		}

		// Apply metrics if available
		if m, ok := metrics[n.Name]; ok {
			node.CPU.Current = m.CPU
			node.Memory.Current = m.Memory

			if node.CPU.Capacity > 0 {
				node.CPU.Percent = float64(m.CPU) / float64(node.CPU.Capacity) * 100
			}
			if node.Memory.Capacity > 0 {
				node.Memory.Percent = float64(m.Memory) / float64(node.Memory.Capacity) * 100
			}
		}

		// Get conditions
		node.Conditions = c.getNodeConditions(n)

		// Count pods on this node
		node.PodCount = c.countPodsOnNode(n.Name)

		// Check for GPU
		node.GPU = c.getGPUInfo(n)

		result = append(result, node)
	}

	return result
}

// getNodeStatus determines if a node is Ready
func (c *Collector) getNodeStatus(node corev1.Node) models.NodeStatus {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return models.NodeStatusReady
			}
			return models.NodeStatusNotReady
		}
	}
	return models.NodeStatusUnknown
}

// getNodeConditions extracts pressure conditions
func (c *Collector) getNodeConditions(node corev1.Node) models.NodeConditions {
	conditions := models.NodeConditions{}
	for _, cond := range node.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
		switch cond.Type {
		case corev1.NodeMemoryPressure:
			conditions.MemoryPressure = true
		case corev1.NodeDiskPressure:
			conditions.DiskPressure = true
		case corev1.NodePIDPressure:
			conditions.PIDPressure = true
		case corev1.NodeNetworkUnavailable:
			conditions.NetworkUnavail = true
		}
	}
	return conditions
}

// countPodsOnNode returns the number of pods running on a node
func (c *Collector) countPodsOnNode(nodeName string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastMetrics == nil {
		return 0
	}

	count := 0
	for _, pod := range c.lastMetrics.Pods {
		if pod.NodeName == nodeName {
			count++
		}
	}
	return count
}

// getGPUInfo extracts GPU information from node labels and capacity
func (c *Collector) getGPUInfo(node corev1.Node) *models.GPUInfo {
	// Check for NVIDIA GPU resources
	gpuQuantity, ok := node.Status.Capacity["nvidia.com/gpu"]
	if !ok {
		// Also check allocatable
		gpuQuantity, ok = node.Status.Allocatable["nvidia.com/gpu"]
		if !ok {
			return nil
		}
	}

	count, _ := gpuQuantity.AsInt64()
	if count == 0 {
		return nil
	}

	info := &models.GPUInfo{
		Count: int(count),
	}

	// Try to get GPU memory from labels
	if memStr, ok := node.Labels["nvidia.com/gpu.memory"]; ok {
		if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil {
			info.MemoryTotal = mem * 1024 * 1024 // Convert MiB to bytes
		}
	}

	return info
}

// fetchPodMetrics fetches pod information and metrics
func (c *Collector) fetchPodMetrics(ctx context.Context) ([]models.Pod, error) {
	// Fetch pod metrics
	podMetricsList, err := c.client.MetricsClient().MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Create a map of metrics by namespace/name
	metricsMap := make(map[string]struct {
		CPU    int64
		Memory int64
	})
	for _, pm := range podMetricsList.Items {
		key := pm.Namespace + "/" + pm.Name
		var cpu, mem int64
		for _, container := range pm.Containers {
			cpu += container.Usage.Cpu().MilliValue()
			mem += container.Usage.Memory().Value()
		}
		metricsMap[key] = struct {
			CPU    int64
			Memory int64
		}{CPU: cpu, Memory: mem}
	}

	// Fetch pod info for status and other details
	podList, err := c.client.Clientset().CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Update namespace list
	nsSet := make(map[string]bool)
	result := make([]models.Pod, 0, len(podList.Items))

	for _, p := range podList.Items {
		nsSet[p.Namespace] = true

		pod := models.Pod{
			Namespace:      p.Namespace,
			Name:           p.Name,
			NodeName:       p.Spec.NodeName,
			Status:         c.getPodStatus(p),
			ContainerCount: len(p.Spec.Containers),
		}

		// Get restart count
		for _, cs := range p.Status.ContainerStatuses {
			pod.RestartCount += cs.RestartCount
		}

		// Apply metrics if available
		key := p.Namespace + "/" + p.Name
		if m, ok := metricsMap[key]; ok {
			pod.CPU = m.CPU
			pod.Memory = m.Memory
		}

		result = append(result, pod)
	}

	// Update namespaces list
	c.mu.Lock()
	c.namespaces = make([]string, 0, len(nsSet))
	for ns := range nsSet {
		c.namespaces = append(c.namespaces, ns)
	}
	sort.Strings(c.namespaces)
	c.mu.Unlock()

	return result, nil
}

// getPodStatus determines the pod status
func (c *Collector) getPodStatus(pod corev1.Pod) models.PodStatus {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		return models.PodStatusRunning
	case corev1.PodPending:
		return models.PodStatusPending
	case corev1.PodSucceeded:
		return models.PodStatusSucceeded
	case corev1.PodFailed:
		return models.PodStatusFailed
	default:
		return models.PodStatusUnknown
	}
}

// calculateAggregates computes cluster-wide totals
func (c *Collector) calculateAggregates(metrics *models.ClusterMetrics) {
	metrics.TotalNodes = len(metrics.Nodes)
	metrics.TotalPods = len(metrics.Pods)

	for _, node := range metrics.Nodes {
		metrics.TotalCPUCapacity += node.CPU.Capacity
		metrics.TotalCPUUsed += node.CPU.Current
		metrics.TotalMemoryCapacity += node.Memory.Capacity
		metrics.TotalMemoryUsed += node.Memory.Current
		metrics.TotalDiskCapacity += node.Disk.Capacity
		metrics.TotalDiskUsed += node.Disk.Current

		// Count CPU cores (capacity in millicores / 1000)
		metrics.TotalCPUCores += int(node.CPU.Capacity / 1000)

		// Count GPUs
		if node.GPU != nil {
			metrics.TotalGPUs += node.GPU.Count
		}

		if node.Status == models.NodeStatusReady {
			metrics.ReadyNodes++
		}
	}
}

// GetNamespaces returns the list of available namespaces
func (c *Collector) GetNamespaces() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.namespaces
}

// GetLastMetrics returns the last collected metrics
func (c *Collector) GetLastMetrics() *models.ClusterMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastMetrics
}

// FormatCPU formats CPU millicores for display
func FormatCPU(millicores int64) string {
	if millicores >= 1000 {
		return fmt.Sprintf("%.1f", float64(millicores)/1000)
	}
	return fmt.Sprintf("%dm", millicores)
}

// FormatMemory formats memory bytes for display
func FormatMemory(bytes int64) string {
	const (
		Ki = 1024
		Mi = Ki * 1024
		Gi = Mi * 1024
	)

	switch {
	case bytes >= Gi:
		return fmt.Sprintf("%.1fGi", float64(bytes)/float64(Gi))
	case bytes >= Mi:
		return fmt.Sprintf("%.0fMi", float64(bytes)/float64(Mi))
	case bytes >= Ki:
		return fmt.Sprintf("%.0fKi", float64(bytes)/float64(Ki))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// FormatPercent formats a percentage for display
func FormatPercent(percent float64) string {
	return fmt.Sprintf("%.1f%%", percent)
}

// ParseResourceQuantity parses a Kubernetes resource quantity string
func ParseResourceQuantity(s string) (int64, error) {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return 0, err
	}
	return q.Value(), nil
}

// SortNodes sorts nodes by the specified field
func SortNodes(nodes []models.Node, field models.SortField, ascending bool) {
	sort.Slice(nodes, func(i, j int) bool {
		var less bool
		switch field {
		case models.SortNodeName:
			less = strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
		case models.SortNodeCPU:
			less = nodes[i].CPU.Percent < nodes[j].CPU.Percent
		case models.SortNodeMemory:
			less = nodes[i].Memory.Percent < nodes[j].Memory.Percent
		case models.SortNodeStatus:
			less = nodes[i].Status < nodes[j].Status
		case models.SortNodePods:
			less = nodes[i].PodCount < nodes[j].PodCount
		default:
			less = nodes[i].Name < nodes[j].Name
		}
		if ascending {
			return less
		}
		return !less
	})
}

// SortPods sorts pods by the specified field
func SortPods(pods []models.Pod, field models.SortField, ascending bool) {
	sort.Slice(pods, func(i, j int) bool {
		var less bool
		switch field {
		case models.SortPodNamespace:
			less = strings.ToLower(pods[i].Namespace) < strings.ToLower(pods[j].Namespace)
		case models.SortPodName:
			less = strings.ToLower(pods[i].Name) < strings.ToLower(pods[j].Name)
		case models.SortPodCPU:
			less = pods[i].CPU < pods[j].CPU
		case models.SortPodMemory:
			less = pods[i].Memory < pods[j].Memory
		case models.SortPodStatus:
			less = pods[i].Status < pods[j].Status
		default:
			less = pods[i].Name < pods[j].Name
		}
		if ascending {
			return less
		}
		return !less
	})
}

// FilterPods filters pods by namespace and system namespace visibility
func FilterPods(pods []models.Pod, namespace string, showSystem bool) []models.Pod {
	result := make([]models.Pod, 0, len(pods))
	for _, pod := range pods {
		// Filter by namespace if specified
		if namespace != "" && pod.Namespace != namespace {
			continue
		}
		// Filter system namespaces if not showing them
		if !showSystem && models.IsSystemNamespace(pod.Namespace) {
			continue
		}
		result = append(result, pod)
	}
	return result
}

// LimitPods returns only the top N pods
func LimitPods(pods []models.Pod, limit int) []models.Pod {
	if len(pods) <= limit {
		return pods
	}
	return pods[:limit]
}
