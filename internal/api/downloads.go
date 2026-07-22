package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/pkg"
)

const agentVersion = "1.0.0"

var platformBinaries = map[string]string{
	"linux-amd64":   "securelens-agent-linux-amd64",
	"linux-arm64":   "securelens-agent-linux-arm64",
	"darwin-amd64":  "securelens-agent-darwin-amd64",
	"darwin-arm64":  "securelens-agent-darwin-arm64",
	"windows-amd64": "securelens-agent-windows-amd64.exe",
}

var platformChecksums = map[string]string{
	"linux-amd64":   "854265223d5dc68bac17da924f2ccde0029f92cb861ddd662d7fd0de137aea58",
	"linux-arm64":   "080e8d5d2891fab0e60098445f0e52517cd54e322330896d1cfa3fbd56b9ddd5",
	"darwin-amd64":  "3835cc8e6d62d8b0c2a25eeda321e3e305d8d0f39c4eef3915671a5eff6c39e9",
	"darwin-arm64":  "fa577b55936a5bc540a77e5036c1392a2f61bbf91ccb23aa5343e09c84285ed7",
	"windows-amd64": "aaa4b56855dfe86a30cb9e130f26b32983c84ea0e2e00d074fcc182fb3e7235d",
}

func (s *Server) downloadAgent(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")

	binaryName, ok := platformBinaries[platform]
	if !ok {
		pkg.Error(w, fmt.Errorf("unsupported platform: %s", platform), http.StatusBadRequest)
		return
	}

	binaryDir := os.Getenv("AGENT_BINARY_DIR")
	if binaryDir == "" {
		binaryDir = "./agent/build"
	}

	binaryPath := filepath.Join(binaryDir, binaryName)

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		pkg.Error(w, fmt.Errorf("binary not found for platform: %s", platform), http.StatusNotFound)
		return
	}

	file, err := os.Open(binaryPath)
	if err != nil {
		pkg.Error(w, fmt.Errorf("failed to open binary: %w", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		pkg.Error(w, fmt.Errorf("failed to stat binary: %w", err), http.StatusInternalServerError)
		return
	}

	downloadName := binaryName
	if strings.HasSuffix(platform, "windows-amd64") {
		w.Header().Set("Content-Type", "application/octet-stream")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", downloadName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.Header().Set("X-Agent-Version", agentVersion)
	w.Header().Set("X-Checksum-SHA256", platformChecksums[platform])

	http.ServeContent(w, r, downloadName, stat.ModTime(), file)
}

func (s *Server) listAgentDownloads(w http.ResponseWriter, r *http.Request) {
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.securelens.ai"
	}

	downloads := []map[string]any{}
	for platform, binary := range platformBinaries {
		parts := strings.Split(platform, "-")
		osName := parts[0]
		arch := parts[1]

		osDisplay := map[string]string{
			"linux":   "Linux",
			"darwin":  "macOS",
			"windows": "Windows",
		}[osName]

		archDisplay := map[string]string{
			"amd64": "x86_64 (Intel/AMD)",
			"arm64": "ARM64 (Apple Silicon/ARM)",
		}[arch]

		downloads = append(downloads, map[string]any{
			"platform":     platform,
			"os":           osDisplay,
			"architecture": archDisplay,
			"filename":     binary,
			"download_url": fmt.Sprintf("%s/api/v1/downloads/agent/%s", baseURL, platform),
			"checksum":     platformChecksums[platform],
		})
	}

	pkg.JSON(w, map[string]any{
		"version":   agentVersion,
		"downloads": downloads,
		"documentation": map[string]string{
			"installation": fmt.Sprintf("%s/docs/features/device-agent", baseURL),
			"downloads":    fmt.Sprintf("%s/docs/downloads", baseURL),
		},
	})
}
