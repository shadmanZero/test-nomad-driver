# Firecracker Task Driver

A [Nomad](https://www.nomadproject.io/) task driver plugin for creating micro-VMs using [AWS Firecracker](https://github.com/firecracker-microvm/firecracker).

## Features

- **Lightweight VMs**: Creates secure micro-VMs using AWS Firecracker
- **OCI Image Support**: Automatically converts OCI/Docker images to Firecracker-compatible rootfs
- **Multiple Container Runtimes**: Works with Docker, Podman, or Skopeo+Buildah
- **Network Integration**: Support for both CNI and static network configuration
- **Resource Management**: CPU and memory allocation control
- **Private Registry Support**: Simple username/password authentication

## Quick Start

### Basic Usage with OCI Images

```hcl
job "example" {
  datacenters = ["dc1"]
  type        = "service"

  group "microvm" {
    task "ubuntu-vm" {
      driver = "firecracker-task-driver"
      
      config {
        # Use any OCI-compatible image
        OCIImage = "ubuntu:22.04"
        
        # Kernel is still required
        KernelImage = "/opt/firecracker/vmlinux"
        
        # VM configuration
        Vcpus = 1
        Mem = 512
        Network = "default"
      }
    }
  }
}
```

### Private Registry Authentication

```hcl
config {
  OCIImage = "registry.example.com/myorg/myapp:latest"
  
  OCIAuth = {
    Username = "myuser"
    Password = "mypass"
  }
  
  KernelImage = "/opt/firecracker/vmlinux"
  Vcpus = 2
  Mem = 1024
}
```

## Configuration

### Task Configuration Options

#### OCI Image Support

- `OCIImage` (string, optional): OCI/Docker image reference (e.g., "ubuntu:22.04", "ghcr.io/org/image:tag")
- `OCIAuth` (block, optional): Authentication for private registries
  - `Username` (string): Registry username
  - `Password` (string): Registry password  

#### Traditional Rootfs

- `BootDisk` (string, optional): Path to pre-built rootfs image file

**Note**: You must specify either `OCIImage` or `BootDisk`, but not both.

#### VM Configuration

- `KernelImage` (string, required): Path to kernel image
- `Vcpus` (number): Number of vCPUs (default: 1)
- `Mem` (number): Memory in MB (default: 512)
- `Cputype` (string): CPU template ("T2" or "C3")
- `DisableHt` (bool): Disable hyperthreading
- `BootOptions` (string): Kernel boot parameters

#### Networking

- `Network` (string): CNI network name for dynamic networking
- `Nic` (block): Static network configuration
  - `Ip` (string): IP address in CIDR format
  - `Gateway` (string): Gateway IP address
  - `Interface` (string): Host tap interface name
  - `Nameservers` (list): DNS servers

#### Storage

- `Disks` (list): Additional disk images (format: "/path/to/disk.ext4:rw")

#### Advanced

- `Firecracker` (string): Path to Firecracker binary
- `Log` (string): Path to log file

## Installation

### Prerequisites

1. **Firecracker**: Install AWS Firecracker
2. **Container Runtime** (one of):
   - **Docker**: Most common, widely available
   - **Podman**: Rootless alternative to Docker
   - **Skopeo + Buildah**: Minimal OCI tools
3. **Filesystem tools**: For rootfs creation

   ```bash
   sudo apt-get install e2fsprogs util-linux
   ```

### Build and Install

```bash
git clone https://github.com/cneira/firecracker-task-driver
cd firecracker-task-driver
go build -o firecracker-task-driver ./main.go
sudo mv firecracker-task-driver /opt/nomad/plugins/
```

### Nomad Configuration

Add to your Nomad client configuration:

```hcl
# In your Nomad agent configuration (e.g. nomad.hcl)

client {
  # This should be set to true for client agents
  enabled = true
}

# Enable the firecracker-task-driver.
# Note: This plugin does not take any configuration options in the `plugin` block.
plugin "firecracker-task-driver" {}
```

## How OCI Image Support Works

The driver automatically detects and uses available container tools:

1. **First Choice: Skopeo + Buildah**
   - Most efficient for rootless operation
   - Direct OCI image manipulation

2. **Second Choice: Podman**
   - Good Docker alternative
   - Built-in OCI support

3. **Third Choice: Docker**
   - Most widely available
   - Fallback option

The process:

1. Pull OCI/Docker images from registries
2. Extract the image layers into a temporary filesystem
3. Convert the extracted filesystem into an ext4 image suitable for Firecracker
4. Clean up temporary files automatically

## Examples

See the [examples](examples/) directory:

- [oci-simple-example.hcl](examples/oci-simple-example.hcl) - Minimal setup
- [oci-example.hcl](examples/oci-example.hcl) - Basic OCI image usage
- [oci-private-registry.hcl](examples/oci-private-registry.hcl) - Private registry authentication

## Tool Requirements

The driver will automatically use the first available tool:

| Tool | Installation | Use Case |
|------|-------------|----------|
| **Skopeo + Buildah** | `sudo apt install skopeo buildah` | Rootless, most efficient |
| **Podman** | `sudo apt install podman` | Docker alternative |
| **Docker** | Standard Docker installation | Most common |

## Limitations

- Requires one of: Docker, Podman, or Skopeo+Buildah
- Requires root access for mounting filesystems during image conversion
- Kernel image must still be provided separately
- OCI image conversion adds startup time (first-time only per image)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

