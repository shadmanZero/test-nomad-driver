job "simple-ocfd" {
  datacenters = ["dc1"]
  type        = "service"

  group "web" {
    task "nginx" {
      driver = "firecracker-task-driver"
      
      config {
        # Use the fully qualified OCI image name for podman
        OCIImage = "docker.io/library/nginx:alpine"
        
        KernelImage = "/root/vms/vmlinux.bin"
        Firecracker = "/usr/local/bin/firecracker"
        
        # This is crucial for giving the VM an IP address
        Network = "default"
        
        Vcpus = 1
        Mem = 256
      }
      
      resources {
        cpu    = 100
        memory = 256
      }
    }
  }
}