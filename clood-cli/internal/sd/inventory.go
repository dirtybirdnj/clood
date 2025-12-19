package sd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CacheTTL is how long inventory cache is valid (1 hour)
const CacheTTL = 1 * time.Hour

// LocalInventory holds all available local models and resources.
type LocalInventory struct {
	Checkpoints []LocalCheckpoint
	LoRAs       []LocalLoRA
	VAEs        []LocalVAE
	Embeddings  []LocalEmbedding
	Hardware    HardwareInfo
	ComfyUIPath string
}

// LocalCheckpoint represents a locally available checkpoint model.
type LocalCheckpoint struct {
	Name      string
	Path      string
	BaseModel string // SD1.5, SDXL, Flux
	Hash      string // SHA256 first 10 chars (CivitAI style)
	Size      int64
	Aliases   []string
}

// LocalLoRA represents a locally available LoRA.
type LocalLoRA struct {
	Name      string
	Path      string
	BaseModel string
	Hash      string
	Size      int64
}

// LocalVAE represents a locally available VAE.
type LocalVAE struct {
	Name string
	Path string
	Hash string
	Size int64
}

// LocalEmbedding represents a textual inversion embedding.
type LocalEmbedding struct {
	Name string
	Path string
}

// HardwareInfo holds detected hardware capabilities.
type HardwareInfo struct {
	GPUName    string
	GPUVendor  string // nvidia, amd, apple
	TotalVRAM  int64
	FreeVRAM   int64
	Backend    string // cuda, rocm, mps, cpu
}

// CheckpointMatch is the result of finding a checkpoint.
type CheckpointMatch struct {
	Checkpoint LocalCheckpoint
	Level      MatchLevel
}

// LoRAMatch is the result of finding a LoRA.
type LoRAMatch struct {
	LoRA  LocalLoRA
	Level MatchLevel
}

// VAEMatch is the result of finding a VAE.
type VAEMatch struct {
	VAE   LocalVAE
	Level MatchLevel
}

// NewLocalInventory creates an empty inventory.
func NewLocalInventory() *LocalInventory {
	return &LocalInventory{}
}

// ScanComfyUI discovers models in a ComfyUI installation.
func (inv *LocalInventory) ScanComfyUI(basePath string) error {
	inv.ComfyUIPath = basePath

	// Standard ComfyUI model directories
	checkpointsDir := filepath.Join(basePath, "models", "checkpoints")
	lorasDir := filepath.Join(basePath, "models", "loras")
	vaesDir := filepath.Join(basePath, "models", "vae")
	embeddingsDir := filepath.Join(basePath, "models", "embeddings")

	// Scan each directory
	if err := inv.scanCheckpoints(checkpointsDir); err != nil {
		// Non-fatal - directory might not exist
	}
	if err := inv.scanLoRAs(lorasDir); err != nil {
		// Non-fatal
	}
	if err := inv.scanVAEs(vaesDir); err != nil {
		// Non-fatal
	}
	if err := inv.scanEmbeddings(embeddingsDir); err != nil {
		// Non-fatal
	}

	return nil
}

func (inv *LocalInventory) scanCheckpoints(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".safetensors" && ext != ".ckpt" && ext != ".pt" {
			return nil
		}

		ckpt := LocalCheckpoint{
			Name:      filepath.Base(path),
			Path:      path,
			Size:      info.Size(),
			BaseModel: inferBaseModelFromName(filepath.Base(path)),
		}

		// Generate hash (first 10 chars of SHA256)
		if hash, err := quickHash(path); err == nil {
			ckpt.Hash = hash
		}

		// Generate aliases
		ckpt.Aliases = generateAliases(ckpt.Name)

		inv.Checkpoints = append(inv.Checkpoints, ckpt)
		return nil
	})
}

func (inv *LocalInventory) scanLoRAs(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".safetensors" && ext != ".pt" {
			return nil
		}

		lora := LocalLoRA{
			Name:      filepath.Base(path),
			Path:      path,
			Size:      info.Size(),
			BaseModel: inferBaseModelFromName(filepath.Base(path)),
		}

		if hash, err := quickHash(path); err == nil {
			lora.Hash = hash
		}

		inv.LoRAs = append(inv.LoRAs, lora)
		return nil
	})
}

func (inv *LocalInventory) scanVAEs(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".safetensors" && ext != ".pt" {
			return nil
		}

		vae := LocalVAE{
			Name: filepath.Base(path),
			Path: path,
			Size: info.Size(),
		}

		if hash, err := quickHash(path); err == nil {
			vae.Hash = hash
		}

		inv.VAEs = append(inv.VAEs, vae)
		return nil
	})
}

func (inv *LocalInventory) scanEmbeddings(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".safetensors" && ext != ".pt" && ext != ".bin" {
			return nil
		}

		emb := LocalEmbedding{
			Name: filepath.Base(path),
			Path: path,
		}

		inv.Embeddings = append(inv.Embeddings, emb)
		return nil
	})
}

// quickHash generates a partial hash (first 10 chars) for model matching.
// Reads only first 1MB to be fast.
func quickHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	// Read first 1MB only for speed
	if _, err := io.CopyN(h, f, 1024*1024); err != nil && err != io.EOF {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil))[:10], nil
}

// inferBaseModelFromName guesses base model from filename.
func inferBaseModelFromName(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "xl") || strings.Contains(name, "sdxl"):
		return "SDXL"
	case strings.Contains(name, "flux"):
		return "Flux"
	case strings.Contains(name, "1.5") || strings.Contains(name, "1_5") || strings.Contains(name, "sd15"):
		return "SD1.5"
	case strings.Contains(name, "2.1") || strings.Contains(name, "2_1"):
		return "SD2.1"
	default:
		return "unknown"
	}
}

// generateAliases creates alternative names for matching.
func generateAliases(name string) []string {
	aliases := []string{name}

	// Remove extension
	base := strings.TrimSuffix(name, filepath.Ext(name))
	aliases = append(aliases, base)

	// Common transformations
	aliases = append(aliases, strings.ReplaceAll(base, "_", " "))
	aliases = append(aliases, strings.ReplaceAll(base, "-", " "))
	aliases = append(aliases, strings.ToLower(base))

	return aliases
}

// FindCheckpoint searches for a matching checkpoint.
func (inv *LocalInventory) FindCheckpoint(spec CheckpointSpec) *CheckpointMatch {
	if spec.Name == "" && spec.Hash == "" {
		return nil
	}

	// 1. Try exact hash match
	if spec.Hash != "" {
		for _, ckpt := range inv.Checkpoints {
			if strings.HasPrefix(ckpt.Hash, spec.Hash) || strings.HasPrefix(spec.Hash, ckpt.Hash) {
				return &CheckpointMatch{Checkpoint: ckpt, Level: MatchExact}
			}
		}
	}

	// 2. Try exact name match
	for _, ckpt := range inv.Checkpoints {
		if strings.EqualFold(ckpt.Name, spec.Name) {
			return &CheckpointMatch{Checkpoint: ckpt, Level: MatchExact}
		}
	}

	// 3. Try alias match
	specLower := strings.ToLower(spec.Name)
	for _, ckpt := range inv.Checkpoints {
		for _, alias := range ckpt.Aliases {
			if strings.Contains(strings.ToLower(alias), specLower) ||
				strings.Contains(specLower, strings.ToLower(alias)) {
				return &CheckpointMatch{Checkpoint: ckpt, Level: MatchSimilar}
			}
		}
	}

	// 4. Try base model family match
	if spec.BaseModel != "" {
		specBase := normalizeBaseModel(spec.BaseModel)
		for _, ckpt := range inv.Checkpoints {
			if normalizeBaseModel(ckpt.BaseModel) == specBase {
				return &CheckpointMatch{Checkpoint: ckpt, Level: MatchPartial}
			}
		}
	}

	return nil
}

// FindLoRA searches for a matching LoRA.
func (inv *LocalInventory) FindLoRA(spec LoRASpec) *LoRAMatch {
	if spec.Name == "" && spec.Hash == "" {
		return nil
	}

	// 1. Exact hash match
	if spec.Hash != "" {
		for _, lora := range inv.LoRAs {
			if strings.HasPrefix(lora.Hash, spec.Hash) || strings.HasPrefix(spec.Hash, lora.Hash) {
				return &LoRAMatch{LoRA: lora, Level: MatchExact}
			}
		}
	}

	// 2. Exact name match
	for _, lora := range inv.LoRAs {
		if strings.EqualFold(lora.Name, spec.Name) {
			return &LoRAMatch{LoRA: lora, Level: MatchExact}
		}
	}

	// 3. Fuzzy name match
	specLower := strings.ToLower(spec.Name)
	for _, lora := range inv.LoRAs {
		loraLower := strings.ToLower(lora.Name)
		if strings.Contains(loraLower, specLower) || strings.Contains(specLower, loraLower) {
			return &LoRAMatch{LoRA: lora, Level: MatchSimilar}
		}
	}

	return nil
}

// FindVAE searches for a matching VAE.
func (inv *LocalInventory) FindVAE(spec VAESpec) *VAEMatch {
	if spec.Name == "" && spec.Hash == "" {
		return nil
	}

	// 1. Exact hash match
	if spec.Hash != "" {
		for _, vae := range inv.VAEs {
			if strings.HasPrefix(vae.Hash, spec.Hash) || strings.HasPrefix(spec.Hash, vae.Hash) {
				return &VAEMatch{VAE: vae, Level: MatchExact}
			}
		}
	}

	// 2. Exact name match
	for _, vae := range inv.VAEs {
		if strings.EqualFold(vae.Name, spec.Name) {
			return &VAEMatch{VAE: vae, Level: MatchExact}
		}
	}

	// 3. Fuzzy match
	specLower := strings.ToLower(spec.Name)
	for _, vae := range inv.VAEs {
		if strings.Contains(strings.ToLower(vae.Name), specLower) {
			return &VAEMatch{VAE: vae, Level: MatchSimilar}
		}
	}

	return nil
}

// FromComfyUIAPI populates inventory from live ComfyUI API.
func (inv *LocalInventory) FromComfyUIAPI(client *Client) error {
	// Get system stats for hardware info
	stats, err := client.GetSystemStats()
	if err == nil {
		if len(stats.Devices) > 0 {
			dev := stats.Devices[0]
			inv.Hardware = HardwareInfo{
				GPUName:   dev.Name,
				TotalVRAM: dev.VRAM,
				FreeVRAM:  dev.VRAMFree,
			}

			// Infer vendor from name
			nameLower := strings.ToLower(dev.Name)
			switch {
			case strings.Contains(nameLower, "nvidia") || strings.Contains(nameLower, "rtx") || strings.Contains(nameLower, "gtx"):
				inv.Hardware.GPUVendor = "nvidia"
				inv.Hardware.Backend = "cuda"
			case strings.Contains(nameLower, "amd") || strings.Contains(nameLower, "radeon"):
				inv.Hardware.GPUVendor = "amd"
				inv.Hardware.Backend = "rocm"
			case strings.Contains(nameLower, "apple") || strings.Contains(nameLower, "m1") || strings.Contains(nameLower, "m2") || strings.Contains(nameLower, "m3") || strings.Contains(nameLower, "m4"):
				inv.Hardware.GPUVendor = "apple"
				inv.Hardware.Backend = "mps"
			default:
				inv.Hardware.Backend = "cpu"
			}
		}
	}

	// Get checkpoints from API
	checkpoints, err := client.GetCheckpoints()
	if err == nil {
		for _, name := range checkpoints {
			inv.Checkpoints = append(inv.Checkpoints, LocalCheckpoint{
				Name:      name,
				BaseModel: inferBaseModelFromName(name),
				Aliases:   generateAliases(name),
			})
		}
	}

	// Get LoRAs from API (if available)
	// ComfyUI doesn't have a direct LoRA list endpoint, but we can try object_info
	info, err := client.GetObjectInfo()
	if err == nil {
		if loraLoader, ok := info["LoraLoader"]; ok {
			if inputs, ok := loraLoader.Input["required"].(map[string]interface{}); ok {
				if loraInfo, ok := inputs["lora_name"].([]interface{}); ok {
					if len(loraInfo) > 0 {
						if loraList, ok := loraInfo[0].([]interface{}); ok {
							for _, l := range loraList {
								if name, ok := l.(string); ok {
									inv.LoRAs = append(inv.LoRAs, LocalLoRA{
										Name:      name,
										BaseModel: inferBaseModelFromName(name),
									})
								}
							}
						}
					}
				}
			}
		}

		// Get VAEs
		if vaeLoader, ok := info["VAELoader"]; ok {
			if inputs, ok := vaeLoader.Input["required"].(map[string]interface{}); ok {
				if vaeInfo, ok := inputs["vae_name"].([]interface{}); ok {
					if len(vaeInfo) > 0 {
						if vaeList, ok := vaeInfo[0].([]interface{}); ok {
							for _, v := range vaeList {
								if name, ok := v.(string); ok {
									inv.VAEs = append(inv.VAEs, LocalVAE{Name: name})
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// Summary returns a human-readable summary of the inventory.
func (inv *LocalInventory) Summary() string {
	var sb strings.Builder

	sb.WriteString("Local Inventory:\n")
	sb.WriteString(fmt.Sprintf("  Checkpoints: %d\n", len(inv.Checkpoints)))
	sb.WriteString(fmt.Sprintf("  LoRAs: %d\n", len(inv.LoRAs)))
	sb.WriteString(fmt.Sprintf("  VAEs: %d\n", len(inv.VAEs)))
	sb.WriteString(fmt.Sprintf("  Embeddings: %d\n", len(inv.Embeddings)))

	if inv.Hardware.GPUName != "" {
		sb.WriteString(fmt.Sprintf("  GPU: %s (%.1fGB VRAM)\n",
			inv.Hardware.GPUName,
			float64(inv.Hardware.TotalVRAM)/(1024*1024*1024)))
	}

	return sb.String()
}

// ToJSON serializes inventory for caching.
func (inv *LocalInventory) ToJSON() ([]byte, error) {
	return json.MarshalIndent(inv, "", "  ")
}

// FromJSON deserializes inventory from cache.
func (inv *LocalInventory) FromJSON(data []byte) error {
	return json.Unmarshal(data, inv)
}

// getCachePath returns the path to the inventory cache file.
func getCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clood", "cache", "inventory.json")
}

// LoadFromCache attempts to load inventory from cache.
// Returns true if cache was valid and loaded, false otherwise.
func (inv *LocalInventory) LoadFromCache() (bool, error) {
	cachePath := getCachePath()

	info, err := os.Stat(cachePath)
	if err != nil {
		return false, nil // Cache doesn't exist
	}

	// Check if cache is expired
	if time.Since(info.ModTime()) > CacheTTL {
		return false, nil // Cache expired
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return false, err
	}

	if err := inv.FromJSON(data); err != nil {
		return false, err
	}

	return true, nil
}

// SaveToCache persists inventory to cache file.
func (inv *LocalInventory) SaveToCache() error {
	cachePath := getCachePath()

	// Ensure cache directory exists
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	data, err := inv.ToJSON()
	if err != nil {
		return fmt.Errorf("serialize inventory: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}

	return nil
}

// InvalidateCache removes the cache file.
func (inv *LocalInventory) InvalidateCache() error {
	cachePath := getCachePath()
	err := os.Remove(cachePath)
	if os.IsNotExist(err) {
		return nil // Already gone
	}
	return err
}

// CacheAge returns how old the cache is, or -1 if no cache exists.
func CacheAge() time.Duration {
	cachePath := getCachePath()
	info, err := os.Stat(cachePath)
	if err != nil {
		return -1
	}
	return time.Since(info.ModTime())
}
