package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/saiyam1814/ing-switch/pkg/server"
	"github.com/spf13/cobra"
)

var uiPort int

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the local web UI for visual migration management",
	Long: `Starts a local web server and opens the ing-switch dashboard in your browser.

The UI provides:
  Detect   — Visual cluster scan with ingress table and complexity badges
  Analyze  — Annotation compatibility matrix and dependency graph
  Migrate  — One-click migration file generation with YAML viewer
  Validate — Post-migration verification with health checks`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUI()
	},
}

func init() {
	uiCmd.Flags().IntVar(&uiPort, "port", 8080, "Port for the local web UI")
	rootCmd.AddCommand(uiCmd)
}

func runUI() error {
	addr := fmt.Sprintf(":%d", uiPort)
	url := fmt.Sprintf("http://localhost%s", addr)

	fmt.Printf("\n  ing-switch UI\n")
	fmt.Printf("  Opening %s\n\n", url)
	fmt.Printf("  Press Ctrl+C to stop\n\n")

	// Open browser
	go openBrowser(url)

	srv := server.NewServer(addr, kubeconfig, kubecontext)
	return srv.Start()
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return
	}

	c := exec.Command(cmd, args...)
	c.Stderr = os.Stderr
	// Ignore error — user can open browser manually
	_ = c.Start()
}
