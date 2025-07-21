job "multi-platform-example" {
  datacenters = ["dc1"]
  type        = "service"

  group "microvm" {
    restart {
      attempts = 1
      mode     = "fail"
    }
    
    task "alpine-vm" {
      driver = "firecracker-task-driver"
      
      config {
        # Multi-platform image (automatically selects correct architecture)
        OCIImage = "alpine:3.18"
        
        # Kernel for the specific architecture
        KernelImage = "/opt/firecracker/vmlinux-x86_64"
        
        # Minimal resources for Alpine
        Vcpus = 1
        Mem = 128
        
        # Use CNI networking
        Network = "firecracker-net"
        
        # Minimal boot options for Alpine
        BootOptions = "console=ttyS0 reboot=k panic=1 pci=off nomodules root=/dev/vda1 rw quiet"
        
        # CPU template for better performance
        Cputype = "T2"
        
        # Disable hyperthreading for security
        DisableHt = true
      }
      
      resources {
        cpu    = 100
        memory = 128
      }
    }
  }
} 