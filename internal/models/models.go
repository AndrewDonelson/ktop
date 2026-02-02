package models

import (
	"time"
)

// NodeStatus represents the health status of a node
type NodeStatus string

const (
	NodeStatusReady    NodeStatus = "Ready"
	NodeStatusNotReady NodeStatus = "NotReady"
	NodeStatusUnknown  NodeStatus = "Unknown"
)

// ResourceUsage holds current and capacity values for a resource
type ResourceUsage struct {
	Current  int64   `json:"current"`  // Current usage in base units (millicores for CPU, bytes for memory)
	Capacity int64   `json:"capacity"` // Total capacity
	Percent  float64 `json:"percent"`  // Usage percentage
}

// GPUInfo holds GPU-related information for a node
type GPUInfo struct {
	Count       int     `json:"count"`
	MemoryUsed  int64   `json:"memoryUsed"`
	MemoryTotal int64   `json:"memoryTotal"`
	Utilization float64 `json:"utilization"`
}

// NodeConditions represents various node pressure conditions
type NodeConditions struct {
	MemoryPressure bool `json:"memoryPressure"`
	DiskPressure   bool `json:"diskPressure"`
	PIDPressure    bool `json:"pidPressure"`
	NetworkUnavail bool `json:"networkUnavailable"`
}

// Node represents a Kubernetes node with its metrics
type Node struct {
	Name       string            `json:"name"`
	Status     NodeStatus        `json:"status"`
	CPU        ResourceUsage     `json:"cpu"`
	Memory     ResourceUsage     `json:"memory"`
	Disk       ResourceUsage     `json:"disk"`
	GPU        *GPUInfo          `json:"gpu,omitempty"`
	PodCount   int               `json:"podCount"`
	Conditions NodeConditions    `json:"conditions"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// PodStatus represents the status of a pod
type PodStatus string

const (
	PodStatusRunning   PodStatus = "Running"
	PodStatusPending   PodStatus = "Pending"
	PodStatusSucceeded PodStatus = "Succeeded"
	PodStatusFailed    PodStatus = "Failed"
	PodStatusUnknown   PodStatus = "Unknown"
)

// Pod represents a Kubernetes pod with its metrics
type Pod struct {
	Namespace      string    `json:"namespace"`
	Name           string    `json:"name"`
	NodeName       string    `json:"nodeName"`
	Status         PodStatus `json:"status"`
	CPU            int64     `json:"cpu"`            // millicores
	Memory         int64     `json:"memory"`         // bytes
	ContainerCount int       `json:"containerCount"`
	RestartCount   int32     `json:"restartCount"`
}

// ClusterInfo holds information about the connected cluster
type ClusterInfo struct {
	Name      string `json:"name"`
	Context   string `json:"context"`
	Server    string `json:"server"`
	Namespace string `json:"namespace,omitempty"` // current namespace if set
}

// ClusterMetrics holds all metrics data for a point in time
type ClusterMetrics struct {
	Timestamp   time.Time   `json:"timestamp"`
	ClusterInfo ClusterInfo `json:"clusterInfo"`
	Nodes       []Node      `json:"nodes"`
	Pods        []Pod       `json:"pods"`
	Error       error       `json:"-"`

	// Aggregate stats
	TotalCPUCapacity    int64 `json:"totalCPUCapacity"`    // millicores
	TotalCPUUsed        int64 `json:"totalCPUUsed"`        // millicores
	TotalCPUCores       int   `json:"totalCPUCores"`       // number of cores
	TotalMemoryCapacity int64 `json:"totalMemoryCapacity"`
	TotalMemoryUsed     int64 `json:"totalMemoryUsed"`
	TotalDiskCapacity   int64 `json:"totalDiskCapacity"`
	TotalDiskUsed       int64 `json:"totalDiskUsed"`
	TotalGPUs           int   `json:"totalGPUs"`
	TotalPods           int   `json:"totalPods"`
	TotalNodes          int   `json:"totalNodes"`
	ReadyNodes          int   `json:"readyNodes"`
}

// SortField represents the field to sort by
type SortField int

const (
	// Node sort fields
	SortNodeName SortField = iota
	SortNodeCPU
	SortNodeMemory
	SortNodeStatus
	SortNodePods

	// Pod sort fields
	SortPodNamespace
	SortPodName
	SortPodCPU
	SortPodMemory
	SortPodStatus
)

// String returns the display name for a sort field
func (s SortField) String() string {
	switch s {
	case SortNodeName:
		return "Name"
	case SortNodeCPU:
		return "CPU"
	case SortNodeMemory:
		return "Memory"
	case SortNodeStatus:
		return "Status"
	case SortNodePods:
		return "Pods"
	case SortPodNamespace:
		return "Namespace"
	case SortPodName:
		return "Name"
	case SortPodCPU:
		return "CPU"
	case SortPodMemory:
		return "Memory"
	case SortPodStatus:
		return "Status"
	default:
		return "Unknown"
	}
}

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewModeSplit ViewMode = iota
	ViewModeNodes
	ViewModePods
)

// AppState holds the current application state
type AppState struct {
	ViewMode        ViewMode
	NodeSortField   SortField
	NodeSortAsc     bool
	PodSortField    SortField
	PodSortAsc      bool
	NamespaceFilter string // empty means all namespaces
	ShowSystem      bool   // show system namespaces
	SelectedNode    int
	SelectedPod     int
	ShowHelp        bool
	LastError       string
}

// DefaultAppState returns the default application state
func DefaultAppState() AppState {
	return AppState{
		ViewMode:        ViewModeSplit,
		NodeSortField:   SortNodeCPU,
		NodeSortAsc:     false, // highest first
		PodSortField:    SortPodCPU,
		PodSortAsc:      false, // highest first
		NamespaceFilter: "",
		ShowSystem:      false,
		SelectedNode:    0,
		SelectedPod:     0,
		ShowHelp:        false,
	}
}

// SystemNamespaces contains namespaces considered "system"
var SystemNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
	"default":         false, // not considered system
}

// IsSystemNamespace checks if a namespace is a system namespace
func IsSystemNamespace(ns string) bool {
	return SystemNamespaces[ns]
}
