# down-pvscope

`down-pvscope` is a Kubernetes tool designed to safely scale down Persistent Volume Claims (PVCs) in Kubernetes clusters. It operates as a Temporal workflow that manages the PVC scaling process while ensuring data safety.

## Overview

This tool helps manage storage resources in Kubernetes clusters by providing a controlled way to reduce the size of Persistent Volume Claims. It's particularly useful for optimizing storage costs and managing cluster resources efficiently.

## Features

- Safe PVC scaling operations in Kubernetes
- Temporal workflow-based execution
- Support for StatefulSet-managed PVCs
- Data integrity protection during scaling
- Rclone integration for data backup/transfer

## Architecture

The project is structured into several key components:

- `cmd/down-pvscope/`: Main application entry point
- `pkg/activities/`: Temporal workflow activities
  - PV/PVC management
  - Rclone operations
  - StatefulSet handling
- `pkg/k8s/`: Kubernetes integration logic
- `pkg/workflows/`: Temporal workflow definitions
- `pkg/util/`: Utility functions and helpers

## Prerequisites

- Kubernetes cluster
- Temporal server
- Access to modify PVCs and StatefulSets
- Rclone (if using backup features)

## Proto Payload

The workflow accepts a proto payload defined in `api/down-pvscope/v1/down-pvscope.proto`:

```protobuf
message Scale {
    string namespace = 1;  // Kubernetes namespace
    string pvc = 2;       // Name of the PVC to scale
    string size = 3;      // Target size for the PVC
    string sts = 4;       // StatefulSet name
}
```

## Development

This is currently a prototype implementation. Contributions and feedback are welcome.
