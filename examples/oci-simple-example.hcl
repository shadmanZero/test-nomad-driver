job "simple-oci" {
  datacenters = ["dc1"]
  type        = "service"

  group "web" {
    task "nginx" {
      driver = "firecracker-task-driver"
      
      config {
        # Simple OCI image usage - just specify the image!
        OCIImage = "nginx:alpine"
        
        # Only kernel is required beyond the image
        KernelImage = "/opt/firecracker/vmlinux"
        
        # Basic VM settings
        Vcpus = 1
        Mem = 256
        Network = "default"
      }
      
      resources {
        cpu    = 100
        memory = 256
      }
    }
  }
} 