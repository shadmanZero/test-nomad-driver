/* Firecracker-task-driver is a task driver for Hashicorp's nomad that allows
 * to create microvms using AWS Firecracker vmm
 * Copyright (C) 2019  Carlos Neira cneirabustos@gmail.com
 *
 * Licensed under the Apache License, Version 2.0 (the "License")
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 */

package firevm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	hclog "github.com/hashicorp/go-hclog"
)

const (
	defaultRootfsSize = "1G"
	defaultTimeout    = "300s"
)

// SimpleOCIManager implements a simplified OCI image manager using external tools
type SimpleOCIManager struct {
	logger  hclog.Logger
	tempDir string
	workDir string
}

// NewSimpleOCIManager creates a new simplified OCI image manager
func NewSimpleOCIManager(logger hclog.Logger, workDir string) (*SimpleOCIManager, error) {
	tempDir, err := os.MkdirTemp("", "firecracker-oci-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &SimpleOCIManager{
		logger:  logger.Named("oci-simple"),
		tempDir: tempDir,
		workDir: workDir,
	}, nil
}

// PullAndCreateRootfs pulls an OCI image and creates a Firecracker-compatible rootfs in one step
func (m *SimpleOCIManager) PullAndCreateRootfs(ctx context.Context, imageRef, outputPath string, auth *OCIAuth) error {
	m.logger.Info("pulling OCI image and creating rootfs", "image", imageRef, "output", outputPath)

	// Try different tools in order of preference: skopeo+buildah, podman, docker
	if m.hasCommand("skopeo") && m.hasCommand("buildah") {
		return m.createRootfsWithSkopeoAndBuildah(ctx, imageRef, outputPath, auth)
	} else if m.hasCommand("podman") {
		return m.createRootfsWithPodman(ctx, imageRef, outputPath, auth)
	} else if m.hasCommand("docker") {
		return m.createRootfsWithDocker(ctx, imageRef, outputPath, auth)
	}

	return fmt.Errorf("no suitable OCI tool found (tried: skopeo+buildah, podman, docker)")
}

// createRootfsWithSkopeoAndBuildah uses skopeo to pull and buildah to extract
func (m *SimpleOCIManager) createRootfsWithSkopeoAndBuildah(ctx context.Context, imageRef, outputPath string, auth *OCIAuth) error {
	extractDir := filepath.Join(m.tempDir, "extract")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}
	defer os.RemoveAll(extractDir)

	// Pull image with skopeo
	ociDir := filepath.Join(m.tempDir, "oci-image")
	pullCmd := exec.CommandContext(ctx, "skopeo", "copy")
	
	if auth != nil {
		if auth.Username != "" && auth.Password != "" {
			pullCmd.Args = append(pullCmd.Args, "--src-creds", auth.Username+":"+auth.Password)
		}
	}
	
	pullCmd.Args = append(pullCmd.Args, "docker://"+imageRef, "oci:"+ociDir)
	
	if output, err := pullCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("skopeo pull failed: %w\nOutput: %s", err, output)
	}

	// Extract with buildah
	extractCmd := exec.CommandContext(ctx, "buildah", "unshare", "sh", "-c",
		fmt.Sprintf("buildah from oci:%s; buildah mount 1 > %s/mountpoint", ociDir, m.tempDir))
	
	if output, err := extractCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("buildah mount failed: %w\nOutput: %s", err, output)
	}

	// Read mount point
	mountData, err := os.ReadFile(filepath.Join(m.tempDir, "mountpoint"))
	if err != nil {
		return fmt.Errorf("failed to read mount point: %w", err)
	}
	mountPoint := strings.TrimSpace(string(mountData))

	// Create ext4 image
	if err := m.createExt4Image(mountPoint, outputPath); err != nil {
		return fmt.Errorf("failed to create ext4 image: %w", err)
	}

	// Cleanup buildah container
	exec.CommandContext(ctx, "buildah", "rm", "1").Run()

	return nil
}

// createRootfsWithPodman uses podman to pull and extract
func (m *SimpleOCIManager) createRootfsWithPodman(ctx context.Context, imageRef, outputPath string, auth *OCIAuth) error {
	containerName := fmt.Sprintf("firecracker-extract-%d", os.Getpid())
	defer func() {
		exec.CommandContext(ctx, "podman", "rm", "-f", containerName).Run()
	}()

	// Create container
	createCmd := exec.CommandContext(ctx, "podman", "create", "--name", containerName)
	
	if auth != nil && auth.Username != "" && auth.Password != "" {
		createCmd.Args = append(createCmd.Args, "--creds", auth.Username+":"+auth.Password)
	}
	
	createCmd.Args = append(createCmd.Args, imageRef, "/bin/true")
	
	if output, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("podman create failed: %w\nOutput: %s", err, output)
	}

	// Export container
	extractDir := filepath.Join(m.tempDir, "extract")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}
	defer os.RemoveAll(extractDir)

	exportCmd := exec.CommandContext(ctx, "podman", "export", containerName)
	tarCmd := exec.CommandContext(ctx, "tar", "-xf", "-", "-C", extractDir)
	
	tarCmd.Stdin, _ = exportCmd.StdoutPipe()
	
	if err := tarCmd.Start(); err != nil {
		return fmt.Errorf("failed to start tar: %w", err)
	}
	
	if err := exportCmd.Run(); err != nil {
		return fmt.Errorf("podman export failed: %w", err)
	}
	
	if err := tarCmd.Wait(); err != nil {
		return fmt.Errorf("tar extraction failed: %w", err)
	}

	return m.createExt4Image(extractDir, outputPath)
}

// createRootfsWithDocker uses docker to pull and extract
func (m *SimpleOCIManager) createRootfsWithDocker(ctx context.Context, imageRef, outputPath string, auth *OCIAuth) error {
	// Login if auth provided
	if auth != nil && auth.Username != "" && auth.Password != "" {
		loginCmd := exec.CommandContext(ctx, "docker", "login", "-u", auth.Username, "--password-stdin")
		loginCmd.Stdin = strings.NewReader(auth.Password)
		if err := loginCmd.Run(); err != nil {
			m.logger.Warn("docker login failed", "error", err)
		}
	}

	containerName := fmt.Sprintf("firecracker-extract-%d", os.Getpid())
	defer func() {
		exec.CommandContext(ctx, "docker", "rm", "-f", containerName).Run()
	}()

	// Create container
	createCmd := exec.CommandContext(ctx, "docker", "create", "--name", containerName, imageRef, "/bin/true")
	if output, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker create failed: %w\nOutput: %s", err, output)
	}

	// Export container
	extractDir := filepath.Join(m.tempDir, "extract")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}
	defer os.RemoveAll(extractDir)

	exportCmd := exec.CommandContext(ctx, "docker", "export", containerName)
	tarCmd := exec.CommandContext(ctx, "tar", "-xf", "-", "-C", extractDir)
	
	tarCmd.Stdin, _ = exportCmd.StdoutPipe()
	
	if err := tarCmd.Start(); err != nil {
		return fmt.Errorf("failed to start tar: %w", err)
	}
	
	if err := exportCmd.Run(); err != nil {
		return fmt.Errorf("docker export failed: %w", err)
	}
	
	if err := tarCmd.Wait(); err != nil {
		return fmt.Errorf("tar extraction failed: %w", err)
	}

	return m.createExt4Image(extractDir, outputPath)
}

// hasCommand checks if a command is available in PATH
func (m *SimpleOCIManager) hasCommand(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// createExt4Image creates an ext4 filesystem image from extracted rootfs
func (m *SimpleOCIManager) createExt4Image(sourceDir, outputPath string) error {
	// Calculate size
	size, err := m.calculateImageSize(sourceDir)
	if err != nil {
		m.logger.Warn("failed to calculate size, using default", "error", err)
		size = defaultRootfsSize
	}

	// Create empty file
	if err := exec.Command("fallocate", "-l", size, outputPath).Run(); err != nil {
		return fmt.Errorf("failed to create image file: %w", err)
	}

	// Format as ext4
	if err := exec.Command("mkfs.ext4", "-F", outputPath).Run(); err != nil {
		return fmt.Errorf("failed to format ext4: %w", err)
	}

	// Mount and copy
	mountDir := filepath.Join(m.tempDir, "mount")
	if err := os.MkdirAll(mountDir, 0755); err != nil {
		return fmt.Errorf("failed to create mount directory: %w", err)
	}

	if err := exec.Command("mount", "-o", "loop", outputPath, mountDir).Run(); err != nil {
		return fmt.Errorf("failed to mount image: %w", err)
	}
	defer exec.Command("umount", mountDir).Run()

	// Copy contents
	if err := exec.Command("cp", "-a", sourceDir+"/.", mountDir+"/").Run(); err != nil {
		return fmt.Errorf("failed to copy contents: %w", err)
	}

	return nil
}

// calculateImageSize calculates required image size with 20% overhead
func (m *SimpleOCIManager) calculateImageSize(sourceDir string) (string, error) {
	output, err := exec.Command("du", "-sb", sourceDir).Output()
	if err != nil {
		return "", err
	}

	var sizeBytes int64
	if _, err := fmt.Sscanf(string(output), "%d", &sizeBytes); err != nil {
		return "", err
	}

	// Add 20% overhead and round up to nearest 100MB
	sizeBytes = sizeBytes + (sizeBytes / 5)
	sizeMB := (sizeBytes / (100 * 1024 * 1024) + 1) * 100
	
	return fmt.Sprintf("%dM", sizeMB), nil
}

// Close cleans up temporary directory
func (m *SimpleOCIManager) Close() error {
	if m.tempDir != "" {
		return os.RemoveAll(m.tempDir)
	}
	return nil
} 