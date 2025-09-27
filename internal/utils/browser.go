package utils

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/smrnjeet222/oxbin/internal/core"
)

// OpenBrowser opens the specified URL in the default browser
func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// GetWebUIURL constructs the URL for viewing a blob in the hosted WebUI
func GetWebUIURL(blobID string) string {
	config := core.DefaultConfig()
	baseURL := strings.TrimSuffix(config.WebAppURL, "/")
	return fmt.Sprintf("%s/%s", baseURL, blobID)
}
