package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// if -v or -verbose has been passed as a flag, the verbose variable will be set to true
// and the logVerbose function will print messages to the console
// if not, the logVerbose function will not print anything.

func init(){
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Parse()
}

var verbose bool // Global verbosity control

func logVerbose(message string) {
	if verbose {
		fmt.Println(message)
	}
}

// main function
func main() {
	if err := installTinyGo(); err != nil {
		fmt.Printf("Installation failed: %s\n", err)
	} else {
		logVerbose("Installation completed successfully.")
	}
}


func installTinyGo() error {
	switch runtime.GOOS {
	case "windows":
		latestVersion, err := getLatestTinyGoVersion()
		if err != nil {
			return err
		}
		return installTinyGoWindows(latestVersion)
	case "linux":
		latestVersion, err := getLatestTinyGoVersion()
		if err != nil {
			return err
		}
		return installTinyGoLinux(latestVersion)
	case "darwin":
		return installTinyGoMacOS()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

// GitHubReleaseInfo holds minimal info about the release
type GitHubReleaseInfo struct {
	TagName string `json:"tag_name"` // Use TagName to capture the version
}

// getLatestTinyGoVersion fetches the latest TinyGo version tag from GitHub releases
func getLatestTinyGoVersion() (string, error) {
	out, err := exec.Command("curl", "-s", "https://api.github.com/repos/tinygo-org/tinygo/releases/latest").Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest TinyGo release info: %w", err)
	}

	var releaseInfo GitHubReleaseInfo
	if err := json.Unmarshal(out, &releaseInfo); err != nil {
		return "", fmt.Errorf("failed to parse release info: %w", err)
	}

	return releaseInfo.TagName, nil
}

func installTinyGoLinux(latestVersion string) error {
	logVerbose("Preparing to install TinyGo on Linux...")

	// finding the pkg manager
	var distrib string
	if _, err := exec.LookPath("apt-get"); err == nil {
		distrib = "debian"
	} else if _, err := exec.LookPath("dnf"); err == nil {
		distrib = "fedora"
	} else if _, err := exec.LookPath("zypper"); err == nil {
		distrib = "opensuse"
	}

	switch distrib{
		case "debian":
			logVerbose("Installing TinyGo on Debian-based Linux...")
			debURL := fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/%s/tinygo_%s_amd64.deb", latestVersion, latestVersion)
			debPath := filepath.Join(os.TempDir(), "tinygo.deb")

			if err := runCommand("curl", "-L", debURL, "-o", debPath); err != nil {
				return fmt.Errorf("failed to download TinyGo: %w", err)
			}

			if err := runCommand("sudo", "dpkg", "-i", debPath); err != nil {
				return fmt.Errorf("failed to install TinyGo with dpkg: %w", err)
			}
		case "fedora":

			// TODO use dnf to install tinygo (DEBUG)
			tarURL := fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/%s/tinygo_%s_linux_amd64.tar.gz", latestVersion, latestVersion)
			tarPath := filepath.Join(os.TempDir(), "tinygo.tar.gz")
		
			logVerbose("Downloading TinyGo for Fedora-based Linux...")
			if err := runCommand("curl", "-L", tarURL, "-o", tarPath); err != nil {
				return fmt.Errorf("failed to download TinyGo: %w", err)
			}
		
			// Provide instructions for manual extraction and PATH update
			logVerbose("Please run the following commands to extract TinyGo and add it to your PATH:")
			fmt.Println("sudo tar -xzf " + tarPath + " -C /usr/local")
			fmt.Println("sudo mv /usr/local/tinygo* /usr/local/tinygo") // Optional: Normalize directory name
			fmt.Printf("echo 'export PATH=\\$PATH:/usr/local/tinygo/bin' >> ~/.bashrc\n")
			fmt.Println("source ~/.bashrc") // or appropriate file like ~/.zshrc for zsh users
		
			// Note: Instructions are provided for the user to run manually.
		default:
			return fmt.Errorf("unsupported distribution")
		
	}
	
	logVerbose("TinyGo installation successful")
	return nil
}

func installTinyGoWindows(latestVersion string) error {
	logVerbose("Preparing to install TinyGo on Windows...")

	// Construct the download URL with the latest version for Windows
	downloadURL := fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/%s/tinygo%s.windows-amd64.zip", latestVersion, latestVersion)
	zipPath := filepath.Join(os.TempDir(), "tinygo.zip")
	extractPath := "C:\\tinygo"

	logVerbose("Downloading TinyGo for Windows...")
	if err := runCommand("curl", "-L", downloadURL, "-o", zipPath); err != nil {
		return err
	}

	logVerbose("Extracting TinyGo...")
	if err := runCommand("powershell", "-Command", "Expand-Archive", "-Path", zipPath, "-DestinationPath", extractPath); err != nil {
		return err
	}

	// handling PATH
	tinyGoBinPath := filepath.Join(extractPath, "bin")
	if err := addPathToSystemEnvironment(tinyGoBinPath); err != nil {
		return err
	}

	logVerbose("TinyGo installation successful on Windows.")
	return nil
}

func addPathToSystemEnvironment(newPath string) error {
	cmd := exec.Command("setx", "PATH", fmt.Sprintf(`%%PATH%%;%s`, newPath), "/M")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add TinyGo to system PATH: %w", err)
	}
	return nil
}

func installTinyGoMacOS() error {
    logVerbose("Setting up TinyGo on macOS via Homebrew...")

    // First, tap the TinyGo tools repository
    tapCmd := exec.Command("brew", "tap", "tinygo-org/tools")
    if err := tapCmd.Run(); err != nil {
        return fmt.Errorf("failed to tap tinygo-org/tools: %w", err)
    }
    logVerbose("Successfully tapped tinygo-org/tools.")

    // Now, install TinyGo using Homebrew
    installCmd := exec.Command("brew", "install", "tinygo")
    if err := installCmd.Run(); err != nil {
        return fmt.Errorf("failed to install TinyGo with Homebrew: %w", err)
    }

    logVerbose("TinyGo installation successful on macOS.")
    return nil
}


func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}



