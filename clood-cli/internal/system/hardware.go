package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// DiskInfo contains information about a disk/filesystem
type DiskInfo struct {
	Mount      string  `json:"mount"`
	Device     string  `json:"device"`
	Filesystem string  `json:"filesystem"`
	TotalGB    float64 `json:"total_gb"`
	UsedGB     float64 `json:"used_gb"`
	FreeGB     float64 `json:"free_gb"`
	UsedPct    float64 `json:"used_pct"`
	IsModels   bool    `json:"is_models"` // True if this disk hosts Ollama models
}

// HardwareInfo contains detected hardware information
type HardwareInfo struct {
	Hostname       string
	OS             string
	Arch           string
	CPUModel       string
	CPUCores       int
	MemoryGB       float64
	GPU            *GPUInfo
	DiskFreeGB     float64   // Legacy: root disk free space
	Disks          []DiskInfo // All detected disks
	ModelsDisk     *DiskInfo  // The disk hosting Ollama models
	ModelsPath     string     // Path to Ollama models directory
	OllamaVRAM     float64    // VRAM available for Ollama (unified memory on Apple Silicon)
}

// GPUInfo contains GPU-specific information
type GPUInfo struct {
	Name     string
	VRAM     float64 // GB
	Type     string  // "apple", "nvidia", "amd", "intel", "none"
	Cores    int     // GPU cores (for Apple Silicon)
}

// DetectHardware gathers hardware information for the local machine
func DetectHardware() (*HardwareInfo, error) {
	info := &HardwareInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	// CPU cores
	info.CPUCores = runtime.NumCPU()

	// OS-specific detection
	switch runtime.GOOS {
	case "darwin":
		detectMacOS(info)
	case "linux":
		detectLinux(info)
	case "windows":
		detectWindows(info)
	default:
		// Basic fallback
		info.CPUModel = runtime.GOARCH
	}

	// Detect all disks and identify the models disk
	detectAllDisks(info)

	return info, nil
}

func detectMacOS(info *HardwareInfo) {
	// Get CPU model
	if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
		info.CPUModel = strings.TrimSpace(string(out))
	}

	// Check if Apple Silicon
	if out, err := exec.Command("sysctl", "-n", "hw.optional.arm64").Output(); err == nil {
		isARM := strings.TrimSpace(string(out)) == "1"
		if isARM {
			detectAppleSilicon(info)
		}
	}

	// Get total memory
	if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
		if bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
			info.MemoryGB = float64(bytes) / (1024 * 1024 * 1024)
		}
	}

	// Get disk free space
	if out, err := exec.Command("df", "-g", "/").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				if free, err := strconv.ParseFloat(fields[3], 64); err == nil {
					info.DiskFreeGB = free
				}
			}
		}
	}

	// For Apple Silicon, VRAM = unified memory (minus some for system)
	if info.GPU != nil && info.GPU.Type == "apple" {
		// Estimate ~75% of unified memory available for GPU tasks
		info.OllamaVRAM = info.MemoryGB * 0.75
	}
}

func detectAppleSilicon(info *HardwareInfo) {
	info.GPU = &GPUInfo{
		Type: "apple",
	}

	// Use system_profiler for detailed info
	out, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		info.GPU.Name = "Apple Silicon GPU"
		return
	}

	output := string(out)

	// Extract chip name
	chipRe := regexp.MustCompile(`Chip Model:\s*(.+)`)
	if matches := chipRe.FindStringSubmatch(output); len(matches) > 1 {
		info.GPU.Name = strings.TrimSpace(matches[1])
	}

	// Extract GPU cores
	coresRe := regexp.MustCompile(`Total Number of Cores:\s*(\d+)`)
	if matches := coresRe.FindStringSubmatch(output); len(matches) > 1 {
		if cores, err := strconv.Atoi(matches[1]); err == nil {
			info.GPU.Cores = cores
		}
	}

	// For Apple Silicon, VRAM is part of unified memory
	// We'll set it based on total memory (detected elsewhere)
	info.GPU.VRAM = info.MemoryGB
}

func detectLinux(info *HardwareInfo) {
	// Get CPU model
	if out, err := exec.Command("cat", "/proc/cpuinfo").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.CPUModel = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	// Get total memory
	if out, err := exec.Command("cat", "/proc/meminfo").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if kb, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
						info.MemoryGB = float64(kb) / (1024 * 1024)
					}
				}
				break
			}
		}
	}

	// Get disk free space
	if out, err := exec.Command("df", "-BG", "/").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				// Remove 'G' suffix
				freeStr := strings.TrimSuffix(fields[3], "G")
				if free, err := strconv.ParseFloat(freeStr, 64); err == nil {
					info.DiskFreeGB = free
				}
			}
		}
	}

	// Try to detect NVIDIA GPU
	detectNvidiaGPU(info)

	// If no GPU, set OllamaVRAM based on system RAM for CPU inference
	if info.GPU == nil && info.MemoryGB > 0 {
		// For CPU-only inference, estimate ~50% of RAM can be used
		info.OllamaVRAM = info.MemoryGB * 0.5
	}
}

func detectWindows(info *HardwareInfo) {
	// Get CPU model via wmic
	if out, err := exec.Command("wmic", "cpu", "get", "name").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 1 {
			// Skip header line "Name"
			info.CPUModel = strings.TrimSpace(lines[1])
		}
	}

	// Fallback: try PowerShell if wmic fails
	if info.CPUModel == "" {
		if out, err := exec.Command("powershell", "-Command", "(Get-WmiObject Win32_Processor).Name").Output(); err == nil {
			info.CPUModel = strings.TrimSpace(string(out))
		}
	}

	// Get total memory via wmic (returns KB)
	if out, err := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 1 {
			if kb, err := strconv.ParseInt(strings.TrimSpace(lines[1]), 10, 64); err == nil {
				info.MemoryGB = float64(kb) / (1024 * 1024)
			}
		}
	}

	// Get disk free space for C: drive
	if out, err := exec.Command("wmic", "logicaldisk", "where", "DeviceID='C:'", "get", "FreeSpace").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 1 {
			if bytes, err := strconv.ParseInt(strings.TrimSpace(lines[1]), 10, 64); err == nil {
				info.DiskFreeGB = float64(bytes) / (1024 * 1024 * 1024)
			}
		}
	}

	// Try to detect NVIDIA GPU (nvidia-smi works on Windows too)
	detectNvidiaGPU(info)

	// If no NVIDIA GPU, set OllamaVRAM based on system RAM for CPU inference
	if info.GPU == nil && info.MemoryGB > 0 {
		// For CPU-only inference, estimate ~50% of RAM can be used
		info.OllamaVRAM = info.MemoryGB * 0.5
	}
}

func detectNvidiaGPU(info *HardwareInfo) {
	out, err := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits").Output()
	if err != nil {
		// No NVIDIA GPU or nvidia-smi not installed
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return
	}

	// Parse first GPU
	parts := strings.Split(lines[0], ", ")
	if len(parts) >= 2 {
		info.GPU = &GPUInfo{
			Name: strings.TrimSpace(parts[0]),
			Type: "nvidia",
		}
		if vram, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
			info.GPU.VRAM = vram / 1024 // Convert MB to GB
			info.OllamaVRAM = info.GPU.VRAM
		}
	}
}

// detectAllDisks detects all mounted filesystems
func detectAllDisks(info *HardwareInfo) {
	switch runtime.GOOS {
	case "darwin":
		detectDisksDarwin(info)
	case "linux":
		detectDisksLinux(info)
	case "windows":
		detectDisksWindows(info)
	}

	// Detect Ollama models path and mark the hosting disk
	info.ModelsPath = getOllamaModelsPath()
	markModelsDisk(info)
}

// getOllamaModelsPath returns the path where Ollama stores models
func getOllamaModelsPath() string {
	// Check OLLAMA_MODELS environment variable first
	if envPath := os.Getenv("OLLAMA_MODELS"); envPath != "" {
		return envPath
	}

	// Default paths by OS
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch runtime.GOOS {
	case "darwin", "linux":
		return home + "/.ollama/models"
	case "windows":
		return home + "\\.ollama\\models"
	}
	return ""
}

// markModelsDisk finds which disk hosts the Ollama models directory
func markModelsDisk(info *HardwareInfo) {
	if info.ModelsPath == "" || len(info.Disks) == 0 {
		return
	}

	// Resolve the models path to an absolute path
	modelsPath := info.ModelsPath
	if absPath, err := realPath(modelsPath); err == nil {
		modelsPath = absPath
	}

	// Find the disk with the longest matching mount point
	var bestMatch *DiskInfo
	bestLen := 0

	for i := range info.Disks {
		disk := &info.Disks[i]
		if strings.HasPrefix(modelsPath, disk.Mount) && len(disk.Mount) > bestLen {
			bestMatch = disk
			bestLen = len(disk.Mount)
		}
	}

	if bestMatch != nil {
		bestMatch.IsModels = true
		info.ModelsDisk = bestMatch
	}
}

// realPath resolves symlinks and returns the absolute path
func realPath(path string) (string, error) {
	// First, get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Then resolve any symlinks
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If eval symlinks fails (e.g., path doesn't exist), return abs path
		return absPath, nil
	}
	return realPath, nil
}

func detectDisksDarwin(info *HardwareInfo) {
	out, err := exec.Command("df", "-g").Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		// Skip non-physical filesystems
		device := fields[0]
		if !strings.HasPrefix(device, "/dev/") {
			continue
		}

		mount := fields[8]
		// Skip system volumes we don't care about
		if strings.HasPrefix(mount, "/System") || strings.HasPrefix(mount, "/private") {
			continue
		}

		total, _ := strconv.ParseFloat(fields[1], 64)
		used, _ := strconv.ParseFloat(fields[2], 64)
		free, _ := strconv.ParseFloat(fields[3], 64)
		usedPct := 0.0
		if total > 0 {
			usedPct = (used / total) * 100
		}

		info.Disks = append(info.Disks, DiskInfo{
			Mount:      mount,
			Device:     device,
			Filesystem: "apfs", // macOS typically uses APFS
			TotalGB:    total,
			UsedGB:     used,
			FreeGB:     free,
			UsedPct:    usedPct,
		})
	}
}

func detectDisksLinux(info *HardwareInfo) {
	out, err := exec.Command("df", "-BG", "--output=source,fstype,size,used,avail,pcent,target").Output()
	if err != nil {
		// Fallback to simpler df
		detectDisksLinuxSimple(info)
		return
	}

	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		device := fields[0]
		// Skip non-physical filesystems
		if !strings.HasPrefix(device, "/dev/") {
			continue
		}

		mount := fields[6]
		// Skip system mounts we don't care about
		if strings.HasPrefix(mount, "/boot") || strings.HasPrefix(mount, "/snap") {
			continue
		}

		fstype := fields[1]
		total, _ := strconv.ParseFloat(strings.TrimSuffix(fields[2], "G"), 64)
		used, _ := strconv.ParseFloat(strings.TrimSuffix(fields[3], "G"), 64)
		free, _ := strconv.ParseFloat(strings.TrimSuffix(fields[4], "G"), 64)
		usedPctStr := strings.TrimSuffix(fields[5], "%")
		usedPct, _ := strconv.ParseFloat(usedPctStr, 64)

		info.Disks = append(info.Disks, DiskInfo{
			Mount:      mount,
			Device:     device,
			Filesystem: fstype,
			TotalGB:    total,
			UsedGB:     used,
			FreeGB:     free,
			UsedPct:    usedPct,
		})
	}
}

func detectDisksLinuxSimple(info *HardwareInfo) {
	out, err := exec.Command("df", "-BG").Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		device := fields[0]
		if !strings.HasPrefix(device, "/dev/") {
			continue
		}

		mount := fields[5]
		total, _ := strconv.ParseFloat(strings.TrimSuffix(fields[1], "G"), 64)
		used, _ := strconv.ParseFloat(strings.TrimSuffix(fields[2], "G"), 64)
		free, _ := strconv.ParseFloat(strings.TrimSuffix(fields[3], "G"), 64)
		usedPctStr := strings.TrimSuffix(fields[4], "%")
		usedPct, _ := strconv.ParseFloat(usedPctStr, 64)

		info.Disks = append(info.Disks, DiskInfo{
			Mount:      mount,
			Device:     device,
			Filesystem: "unknown",
			TotalGB:    total,
			UsedGB:     used,
			FreeGB:     free,
			UsedPct:    usedPct,
		})
	}
}

func detectDisksWindows(info *HardwareInfo) {
	out, err := exec.Command("wmic", "logicaldisk", "get", "DeviceID,FileSystem,FreeSpace,Size").Output()
	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		device := fields[0]
		fstype := fields[1]
		free, _ := strconv.ParseInt(fields[2], 10, 64)
		total, _ := strconv.ParseInt(fields[3], 10, 64)

		freeGB := float64(free) / (1024 * 1024 * 1024)
		totalGB := float64(total) / (1024 * 1024 * 1024)
		usedGB := totalGB - freeGB
		usedPct := 0.0
		if totalGB > 0 {
			usedPct = (usedGB / totalGB) * 100
		}

		info.Disks = append(info.Disks, DiskInfo{
			Mount:      device + "\\",
			Device:     device,
			Filesystem: fstype,
			TotalGB:    totalGB,
			UsedGB:     usedGB,
			FreeGB:     freeGB,
			UsedPct:    usedPct,
		})
	}
}

// ModelsDiskFreeGB returns free space on the disk hosting Ollama models
func (h *HardwareInfo) ModelsDiskFreeGB() float64 {
	if h.ModelsDisk != nil {
		return h.ModelsDisk.FreeGB
	}
	return h.DiskFreeGB // Fallback to legacy root disk
}

// ModelFits returns true if the given model size (in billions) fits in available VRAM
func (h *HardwareInfo) ModelFits(sizeB float64) bool {
	// Rough estimate: model needs ~0.5-1GB per billion parameters (quantized)
	// Using 0.6 as a conservative estimate for Q4 quantization
	requiredVRAM := sizeB * 0.6
	return h.OllamaVRAM >= requiredVRAM
}

// RecommendedModels returns model sizes that would work well on this hardware
func (h *HardwareInfo) RecommendedModels() []string {
	var models []string

	if h.OllamaVRAM >= 48 {
		models = append(models, "qwen2.5-coder:32b", "llama3.1:70b")
	}
	if h.OllamaVRAM >= 24 {
		models = append(models, "codestral:22b", "qwen2.5-coder:14b")
	}
	if h.OllamaVRAM >= 12 {
		models = append(models, "qwen2.5-coder:7b", "llama3.1:8b", "deepseek-coder:6.7b")
	}
	if h.OllamaVRAM >= 6 {
		models = append(models, "qwen2.5-coder:3b", "phi3:3.8b")
	}
	if h.OllamaVRAM >= 2 {
		models = append(models, "qwen2.5-coder:1.5b", "tinyllama")
	}

	return models
}

// Summary returns a human-readable summary
func (h *HardwareInfo) Summary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Host:     %s\n", h.Hostname))
	sb.WriteString(fmt.Sprintf("OS:       %s/%s\n", h.OS, h.Arch))
	sb.WriteString(fmt.Sprintf("CPU:      %s (%d cores)\n", h.CPUModel, h.CPUCores))
	sb.WriteString(fmt.Sprintf("Memory:   %.1f GB\n", h.MemoryGB))

	if h.GPU != nil {
		if h.GPU.Type == "apple" {
			sb.WriteString(fmt.Sprintf("GPU:      %s", h.GPU.Name))
			if h.GPU.Cores > 0 {
				sb.WriteString(fmt.Sprintf(" (%d-core)", h.GPU.Cores))
			}
			sb.WriteString(" [unified memory]\n")
		} else {
			sb.WriteString(fmt.Sprintf("GPU:      %s (%.1f GB VRAM)\n", h.GPU.Name, h.GPU.VRAM))
		}
	} else {
		sb.WriteString(fmt.Sprintf("GPU:      None (CPU inference, ~%.0f GB usable)\n", h.OllamaVRAM))
	}

	return sb.String()
}

// DiskSummary returns a formatted summary of all disks
func (h *HardwareInfo) DiskSummary() string {
	var sb strings.Builder

	if len(h.Disks) == 0 {
		sb.WriteString(fmt.Sprintf("  /  %.1f GB free\n", h.DiskFreeGB))
		return sb.String()
	}

	for _, disk := range h.Disks {
		marker := " "
		if disk.IsModels {
			marker = "*"
		}
		bar := renderDiskBar(disk.UsedPct)
		sb.WriteString(fmt.Sprintf("  %s%-20s %s %5.1f/%5.1f GB (%.0f%% used)\n",
			marker, disk.Mount, bar, disk.FreeGB, disk.TotalGB, disk.UsedPct))
	}

	if h.ModelsPath != "" {
		sb.WriteString(fmt.Sprintf("\n  * Models: %s\n", h.ModelsPath))
		if h.ModelsDisk != nil {
			sb.WriteString(fmt.Sprintf("    Headroom: %.1f GB free for new models\n", h.ModelsDisk.FreeGB))
		}
	}

	return sb.String()
}

// renderDiskBar creates a visual bar for disk usage
func renderDiskBar(usedPct float64) string {
	const barWidth = 10
	filled := int((usedPct / 100) * barWidth)
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return "[" + bar + "]"
}

// JSON returns the hardware info as JSON-compatible map
func (h *HardwareInfo) JSON() map[string]interface{} {
	result := map[string]interface{}{
		"hostname":       h.Hostname,
		"os":             h.OS,
		"arch":           h.Arch,
		"cpu_model":      h.CPUModel,
		"cpu_cores":      h.CPUCores,
		"memory_gb":      h.MemoryGB,
		"disk_free_gb":   h.DiskFreeGB,
		"ollama_vram_gb": h.OllamaVRAM,
		"models_path":    h.ModelsPath,
	}

	if h.GPU != nil {
		result["gpu"] = map[string]interface{}{
			"name":  h.GPU.Name,
			"type":  h.GPU.Type,
			"vram":  h.GPU.VRAM,
			"cores": h.GPU.Cores,
		}
	}

	// Add all disks
	if len(h.Disks) > 0 {
		var disks []map[string]interface{}
		for _, disk := range h.Disks {
			disks = append(disks, map[string]interface{}{
				"mount":      disk.Mount,
				"device":     disk.Device,
				"filesystem": disk.Filesystem,
				"total_gb":   disk.TotalGB,
				"used_gb":    disk.UsedGB,
				"free_gb":    disk.FreeGB,
				"used_pct":   disk.UsedPct,
				"is_models":  disk.IsModels,
			})
		}
		result["disks"] = disks
	}

	// Add models disk headroom
	if h.ModelsDisk != nil {
		result["models_disk_free_gb"] = h.ModelsDisk.FreeGB
	}

	return result
}
