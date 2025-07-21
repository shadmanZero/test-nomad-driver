job "private-registry-example" {
  datacenters = ["dc1"]
  type        = "service"

  group "microvm" {
    restart {
      attempts = 2
      mode     = "fail"
    }
    
    task "custom-app" {
      driver = "firecracker-task-driver"
      
      config {
        # Private registry OCI image
        OCIImage = "registry.example.com/myorg/myapp:v1.2.3"
        
        # Authentication for private registry
        OCIAuth = {
          Username = "myusername"
          Password = "mypassword"
          # Alternative: use registry token
          # RegistryToken = "ghp_xxxxxxxxxxxx"
        }
        
        # Kernel configuration
        KernelImage = "/opt/firecracker/vmlinux"
        
        # VM resources
        Vcpus = 1
        Mem = 256
        
        # Static network configuration
        Nic = {
          Ip = "172.16.0.10/24"
          Gateway = "172.16.0.1"
          Interface = "tap0"
          Nameservers = ["8.8.8.8", "8.8.4.4"]
        }
        
        # Boot options optimized for containers
        BootOptions = "console=ttyS0 reboot=k panic=1 pci=off nomodules root=/dev/vda1 rw init=/sbin/init"
        
        # Enable detailed logging
        Log = "/var/log/firecracker/custom-app.log"
      }
      
      resources {
        cpu    = 100
        memory = 256
      }
    }
  }
} 