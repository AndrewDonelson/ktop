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
	Current  int64   // Current usage in base units (millicores for CPU, bytes for memory)
	Capacity int64   // Total capacity
	Percent  float64 // Usage percentage
}

// GPUInfo holds GPU-related information for a node
type GPUInfo struct {
	Count       int
	MemoryUsed  int64
	MemoryTotal int64
	Utilization float64
}

// NodeConditions represents various node pressure conditions
type NodeConditions struct {
	MemoryPressure bool
	DiskPressure   bool
	PIDPressure    bool
	NetworkUnavail bool
}

// Node represents a Kubernetes node with its metrics
type Node struct {
	Name       string
	Status     NodeStatus
	CPU        ResourceUsage
	Memory     ResourceUsage
	Disk       ResourceUsage
	GPU        *GPUInfo
	PodCount   int
	Conditions NodeConditions
	Labels     map[string]string
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
	Namespace      string
	Name           string
	NodeName       string
	Status         PodStatus
	CPU            int64 // millicores
	Memory         int64 // bytes
	ContainerCount int
	RestartCount   int32
}

// ClusterInfo holds information about the connected cluster
type ClusterInfo struct {
	Name      string
	Context   string
	Server    string
	Namespace string // current namespace if set
}

// ClusterMetrics holds all metrics data for a point in time
type ClusterMetrics struct {
	Timestamp   time.Time
	ClusterInfo ClusterInfo
	Nodes       []Node
	Pods        []Pod
	Error       error

	// Aggregate stats
	TotalCPUCapacity    int64 // millicores
	TotalCPUUsed        int64 // millicores
	TotalCPUCores       int   // number of cores
	TotalMemoryCapacity int64
	TotalMemoryUsed     int64
	TotalDiskCapacity   int64
	TotalDiskUsed       int64
	TotalGPUs           int
	TotalPods           int
	TotalNodes          int
	ReadyNodes          int
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
