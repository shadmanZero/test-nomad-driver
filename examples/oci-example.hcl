job "oci-example" {
  datacenters = ["dc1"]
  type        = "service"

  group "microvm" {
    restart {
      attempts = 0
      mode     = "fail"
    }
    
    task "ubuntu-vm" {
      driver = "firecracker-task-driver"
      
      config {
        # Use OCI image instead of BootDisk
        OCIImage = "ubuntu:22.04"
        
        # Kernel configuration (still required)
        KernelImage = "/opt/firecracker/vmlinux"
        
        # VM resources
        Vcpus = 2
        Mem = 512
        
        # Network configuration
        Network = "default"
        
        # Optional: Authentication for private registries
        # OCIAuth = {
        #   Username = "myuser"
        #   Password = "mypass"
        # }
        
        # Boot options
        BootOptions = "console=ttyS0 reboot=k panic=1 pci=off nomodules root=/dev/vda1 rw"
        
        # Optional: Additional disks
        # Disks = [ "/path/to/additional/disk.ext4:rw" ]
        
        # Optional: Firecracker binary path
        # Firecracker = "/usr/bin/firecracker"
        
        # Optional: Enable logging
        # Log = "/tmp/firecracker.log"
      }
      
      # Resource allocation
      resources {
        cpu    = 200  # 200 MHz
        memory = 512  # 512 MB
      }
    }
  }
} 