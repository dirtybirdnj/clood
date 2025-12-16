package system

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// HardwareInfo contains detected hardware information
type HardwareInfo struct {
	Hostname    string
	OS          string
	Arch        string
	CPUModel    string
	CPUCores    int
	MemoryGB    float64
	GPU         *GPUInfo
	DiskFreeGB  float64
	OllamaVRAM  float64 // VRAM available for Ollama (unified memory on Apple Silicon)
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
	default:
		// Basic fallback
		info.CPUModel = runtime.GOARCH
	}

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
		sb.WriteString("GPU:      None detected\n")
	}

	sb.WriteString(fmt.Sprintf("Disk:     %.1f GB free\n", h.DiskFreeGB))
	sb.WriteString(fmt.Sprintf("Ollama:   ~%.1f GB available for models\n", h.OllamaVRAM))

	return sb.String()
}

// JSON returns the hardware info as JSON-compatible map
func (h *HardwareInfo) JSON() map[string]interface{} {
	result := map[string]interface{}{
		"hostname":    h.Hostname,
		"os":          h.OS,
		"arch":        h.Arch,
		"cpu_model":   h.CPUModel,
		"cpu_cores":   h.CPUCores,
		"memory_gb":   h.MemoryGB,
		"disk_free_gb": h.DiskFreeGB,
		"ollama_vram_gb": h.OllamaVRAM,
	}

	if h.GPU != nil {
		result["gpu"] = map[string]interface{}{
			"name":  h.GPU.Name,
			"type":  h.GPU.Type,
			"vram":  h.GPU.VRAM,
			"cores": h.GPU.Cores,
		}
	}

	return result
}
