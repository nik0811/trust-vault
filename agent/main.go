package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/securelens/securelens-agent/api"
	"github.com/securelens/securelens-agent/config"
	"github.com/securelens/securelens-agent/scanner"
	"github.com/spf13/cobra"
)

var (
	version   = "1.0.0"
	cfgFile   string
	verbose   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "securelens-agent",
		Short:   "SecureLens Device Agent - Scan local files for PII/sensitive data",
		Version: version,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.securelens/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(scanCmd())
	rootCmd.AddCommand(daemonCmd())
	rootCmd.AddCommand(statusCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	var apiKey, apiURL string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the agent with SecureLens API credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if apiKey == "" {
				return fmt.Errorf("--api-key is required")
			}
			if apiURL == "" {
				apiURL = "https://api.securelens.ai"
			}

			hostname, _ := os.Hostname()
			fmt.Printf("Initializing SecureLens Agent on %s...\n", hostname)

			cfg := &config.Config{
				APIKey:   apiKey,
				APIURL:   apiURL,
				Hostname: hostname,
			}

			client := api.NewClient(cfg)
			resp, err := client.Register(hostname)
			if err != nil {
				return fmt.Errorf("registration failed: %w", err)
			}

			cfg.AgentID = resp.AgentID
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Println()
			fmt.Println("✓ Agent registered successfully!")
			fmt.Printf("  Agent ID: %s\n", resp.AgentID)
			fmt.Printf("  Config saved to: %s\n", config.DefaultConfigPath())
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  securelens-agent scan /path/to/scan")
			fmt.Println("  securelens-agent daemon --interval 1h")

			return nil
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "SecureLens API key (required)")
	cmd.Flags().StringVar(&apiURL, "api-url", "https://api.securelens.ai", "SecureLens API URL")
	cmd.MarkFlagRequired("api-key")

	return cmd
}

func scanCmd() *cobra.Command {
	var exclude []string
	var upload bool
	var maxSize int64

	cmd := &cobra.Command{
		Use:   "scan [paths...]",
		Short: "Scan directories for PII/sensitive data",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			paths := args
			for i, p := range paths {
				if abs, err := filepath.Abs(p); err == nil {
					paths[i] = abs
				}
			}

			fmt.Printf("SecureLens Agent v%s\n", version)
			fmt.Println(strings.Repeat("─", 50))
			fmt.Printf("Scanning %d path(s)...\n\n", len(paths))

			s := scanner.New(
				scanner.WithExclude(exclude),
				scanner.WithMaxFileSize(maxSize*1024*1024),
				scanner.WithVerbose(verbose),
				scanner.WithProgress(func(file string, findings int) {
					if verbose {
						if findings > 0 {
							fmt.Printf("  %s (%d findings)\n", file, findings)
						}
					}
				}),
			)

			result, err := s.Scan(paths)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			printFindings(result)

			if upload && cfg.AgentID != "" {
				fmt.Println("\nUploading results to SecureLens...")
				client := api.NewClient(cfg)
				resp, err := client.Report(result, paths)
				if err != nil {
					fmt.Printf("⚠ Upload failed: %v\n", err)
				} else {
					fmt.Printf("✓ Results uploaded successfully!\n")
					if resp.ViewURL != "" {
						fmt.Printf("  View results at: %s\n", resp.ViewURL)
					}
				}
			}

			cfg.LastScan = time.Now().Format(time.RFC3339)
			cfg.Save(cfgFile)

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&exclude, "exclude", []string{"*.gz", "*.zip", "*.tar"}, "patterns to exclude")
	cmd.Flags().BoolVar(&upload, "upload", true, "upload results to SecureLens")
	cmd.Flags().Int64Var(&maxSize, "max-size", 50, "max file size in MB")

	return cmd
}

func daemonCmd() *cobra.Command {
	var interval string
	var paths []string

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run as a daemon with scheduled scans",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			duration, err := time.ParseDuration(interval)
			if err != nil {
				return fmt.Errorf("invalid interval: %w", err)
			}

			if len(paths) == 0 {
				paths = []string{"/var/log", "/etc", "/home"}
			}

			fmt.Printf("SecureLens Agent Daemon v%s\n", version)
			fmt.Println(strings.Repeat("─", 50))
			fmt.Printf("Scan interval: %s\n", interval)
			fmt.Printf("Paths: %s\n", strings.Join(paths, ", "))
			fmt.Println()

			client := api.NewClient(cfg)

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			ticker := time.NewTicker(duration)
			defer ticker.Stop()

			runScan := func() {
				fmt.Printf("[%s] Starting scheduled scan...\n", time.Now().Format("2006-01-02 15:04:05"))
				s := scanner.New(scanner.WithVerbose(verbose))
				result, err := s.Scan(paths)
				if err != nil {
					fmt.Printf("  ⚠ Scan error: %v\n", err)
					return
				}

				fmt.Printf("  Scanned %d files, found %d findings\n", result.FilesScanned, len(result.Findings))

				if len(result.Findings) > 0 {
					resp, err := client.Report(result, paths)
					if err != nil {
						fmt.Printf("  ⚠ Upload failed: %v\n", err)
					} else {
						fmt.Printf("  ✓ Results uploaded (Report ID: %s)\n", resp.ReportID)
					}
				}

				client.Heartbeat()
			}

			runScan()

			for {
				select {
				case <-ticker.C:
					runScan()
				case <-sigChan:
					fmt.Println("\nShutting down...")
					return nil
				}
			}
		},
	}

	cmd.Flags().StringVar(&interval, "interval", "1h", "scan interval (e.g., 30m, 1h, 24h)")
	cmd.Flags().StringSliceVar(&paths, "paths", nil, "paths to scan (default: /var/log, /etc, /home)")

	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show agent status and recent scan results",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			fmt.Printf("SecureLens Agent v%s\n", version)
			fmt.Println(strings.Repeat("─", 50))
			fmt.Printf("Agent ID:  %s\n", cfg.AgentID)
			fmt.Printf("API URL:   %s\n", cfg.APIURL)
			fmt.Printf("Hostname:  %s\n", cfg.Hostname)
			fmt.Printf("Last Scan: %s\n", cfg.LastScan)
			fmt.Println()

			if cfg.AgentID != "" {
				client := api.NewClient(cfg)
				status, err := client.Status()
				if err != nil {
					fmt.Printf("⚠ Could not fetch remote status: %v\n", err)
				} else {
					fmt.Println("Remote Status:")
					fmt.Printf("  Status:        %s\n", status.Status)
					fmt.Printf("  Total Scans:   %d\n", status.TotalScans)
					fmt.Printf("  Total Findings: %d\n", status.Findings)
				}
			}

			return nil
		},
	}
}

func printFindings(result *scanner.ScanResult) {
	severityOrder := map[string]int{"CRITICAL": 0, "HIGH": 1, "MEDIUM": 2, "LOW": 3}
	severityColor := map[string]string{
		"CRITICAL": "\033[31m",
		"HIGH":     "\033[33m",
		"MEDIUM":   "\033[36m",
		"LOW":      "\033[37m",
	}
	reset := "\033[0m"

	critical, high, medium, low := 0, 0, 0, 0
	for _, f := range result.Findings {
		switch f.Severity {
		case "CRITICAL":
			critical++
		case "HIGH":
			high++
		case "MEDIUM":
			medium++
		case "LOW":
			low++
		}
	}

	sorted := make([]scanner.Finding, len(result.Findings))
	copy(sorted, result.Findings)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if severityOrder[sorted[i].Severity] > severityOrder[sorted[j].Severity] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	for _, f := range sorted {
		color := severityColor[f.Severity]
		fmt.Printf("%s[%s]%s %s:%d - %s found: %s\n",
			color, f.Severity, reset,
			f.FilePath, f.LineNumber,
			f.PIIType, f.Masked)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Scan complete: %s files scanned, %d findings\n",
		formatNumber(result.FilesScanned), len(result.Findings))
	fmt.Printf("  Critical: %d | High: %d | Medium: %d | Low: %d\n",
		critical, high, medium, low)
	fmt.Printf("  Duration: %s\n", result.EndTime.Sub(result.StartTime).Round(time.Millisecond))

	if len(result.Errors) > 0 {
		fmt.Printf("  Errors: %d\n", len(result.Errors))
	}
}

func formatNumber(n int) string {
	str := fmt.Sprintf("%d", n)
	if n < 1000 {
		return str
	}
	var result []string
	for i := len(str); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		result = append([]string{str[start:i]}, result...)
	}
	return strings.Join(result, ",")
}
