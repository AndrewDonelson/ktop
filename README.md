# ktop

A terminal UI (TUI) monitoring tool for Kubernetes clusters, similar to `htop` for Linux or `nvtop` for GPUs.

![ktop screenshot](docs/screenshot.png)

## Features

- **Real-time monitoring** - Updates metrics every 2 seconds (configurable)
- **Node-level visibility** - CPU, memory, disk usage per node with capacity percentages
- **Pod-level visibility** - Resource usage for running pods with sortable columns
- **GPU support** - Display GPU count and memory if available (NVIDIA)
- **Status indicators** - Visual feedback on node health and resource pressure
- **Lightweight** - Single binary, minimal dependencies
- **Interactive** - Keyboard controls for sorting, filtering, and navigation

## Installation

### From Source

Requires Go 1.25 or later.

```bash
# Clone the repository
git clone https://github.com/nlaak/ktop.git
cd ktop

# Build
make build

# Install to /usr/local/bin
sudo make install-system

# Or install to GOPATH/bin
make install
```

### Pre-built Binaries

Download from the [releases page](https://github.com/nlaak/ktop/releases).

```bash
# Linux AMD64
curl -LO https://github.com/nlaak/ktop/releases/latest/download/ktop-linux-amd64
chmod +x ktop-linux-amd64
sudo mv ktop-linux-amd64 /usr/local/bin/ktop

# macOS ARM64 (Apple Silicon)
curl -LO https://github.com/nlaak/ktop/releases/latest/download/ktop-darwin-arm64
chmod +x ktop-darwin-arm64
sudo mv ktop-darwin-arm64 /usr/local/bin/ktop
```

## Requirements

- Kubernetes cluster with **metrics-server** installed
- Valid kubeconfig file
- Network access to the Kubernetes API server

### Installing metrics-server

**Standard Kubernetes:**
```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

**MicroK8s:**
```bash
microk8s enable metrics-server
```

**k3s:**
```bash
# metrics-server is included by default
```

## Usage

```bash
# Basic usage (uses default kubeconfig)
ktop

# Specify kubeconfig
ktop -kubeconfig /path/to/kubeconfig

# Use specific context
ktop -context my-cluster

# Faster refresh rate
ktop -refresh-interval 1s

# Show system namespaces
ktop -all-namespaces

# Show more pods
ktop -top-pods 50
```

### Command-line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-kubeconfig` | `~/.kube/config` | Path to kubeconfig file |
| `-context` | current | Kubernetes context to use |
| `-refresh-interval` | `2s` | Metrics refresh interval |
| `-timeout` | `10s` | API call timeout |
| `-top-pods` | `30` | Number of top pods to display |
| `-all-namespaces` | `false` | Include system namespaces |
| `-version` | - | Show version |
| `-help` | - | Show help |

## Keyboard Controls

| Key | Action |
|-----|--------|
| `q` | Quit |
| `r` | Force refresh |
| `s` | Sort nodes (cycle: name → CPU → memory → status → pods) |
| `p` | Sort pods (cycle: namespace → name → CPU → memory) |
| `f` / `n` | Cycle namespace filter |
| `t` | Toggle view mode (split / nodes only / pods only) |
| `a` | Toggle system namespaces visibility |
| `Tab` | Switch focus between nodes and pods tables |
| `?` | Show help |
| `Esc` | Clear namespace filter |
| `↑` / `↓` | Navigate selection |

## UI Layout

```
┌─────────────────────────────────────────────────────────────────────────┐
│ ktop - cluster-name (context)   Nodes: 3/3  CPU: 25.0%  Mem: 45.2%     │
├─────────────────────────────────────────────────────────────────────────┤
│ NODES (sort: CPU ↓)                                                     │
├──────────┬──────────┬──────────┬──────────┬──────────┬─────────────────┤
│ NODE     │ STATUS   │ CPU      │ CPU%     │ MEMORY   │ MEM%   PODS GPU │
├──────────┼──────────┼──────────┼──────────┼──────────┼─────────────────┤
│ cqai     │ Ready    │ 2500m    │ 20.0%    │ 8.0Gi    │ 25.0%   24   2  │
│ ox       │ Ready    │ 1200m    │ 10.0%    │ 4.0Gi    │ 12.0%   18   0  │
│ mule     │ Ready    │ 800m     │ 6.0%     │ 3.0Gi    │ 9.0%    15   0  │
├─────────────────────────────────────────────────────────────────────────┤
│ PODS (top 30 by CPU ↓) [filter: all]                                   │
├───────────────────┬────────────────────┬──────────┬─────────────────────┤
│ NAMESPACE         │ POD                │ STATUS   │ CPU    MEMORY  ... │
├───────────────────┼────────────────────┼──────────┼─────────────────────┤
│ gpu-operator      │ gpu-feature-...    │ Running  │ 45m    128Mi       │
│ kube-system       │ coredns-...        │ Running  │ 23m    45Mi        │
│ default           │ my-app-deployment  │ Running  │ 156m   512Mi       │
├─────────────────────────────────────────────────────────────────────────┤
│ q quit  r refresh  s sort nodes  p pod sort  f/n namespace  t toggle   │
└─────────────────────────────────────────────────────────────────────────┘
```

## Color Coding

| Usage Level | Color | Threshold |
|-------------|-------|-----------|
| Healthy | Green | 0-50% |
| Warning | Yellow | 50-80% |
| Critical | Red | 80%+ |

| Status | Color |
|--------|-------|
| Ready | Green |
| NotReady | Red |
| Running | Green |
| Pending | Yellow |
| Failed | Red |
| System namespace | Cyan |

## Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build for specific platforms
make build-linux    # Linux amd64 & arm64
make build-darwin   # macOS amd64 & arm64
make build-windows  # Windows amd64
```

## Development

```bash
# Run directly
make run

# Run with development flags
make run-dev

# Format code
make fmt

# Run tests
make test

# Lint
make lint
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [tview](https://github.com/rivo/tview) - Terminal UI library
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes Go client
- Inspired by [htop](https://htop.dev/), [nvtop](https://github.com/Syllo/nvtop), and [k9s](https://k9scli.io/)
