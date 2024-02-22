package main

import (
	"archive/tar"
	"bufio"
    "compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// downloadFile downloads a URL to a local file.
func downloadFile(URL, fileName string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// verifyChecksum compares the SHA256 checksum of the file against the expected hash.
func verifyChecksum(filePath, expectedHash string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return false
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Println("Error calculating hash:", err)
		return false
	}

	computedHash := hex.EncodeToString(hash.Sum(nil))
	return strings.EqualFold(computedHash, expectedHash)
}

// readChecksum retrieves the checksum from a .sha256 file.
func readChecksum(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Assuming the checksum file format is "HASH filename"
	parts := strings.Fields(string(content))
	if len(parts) < 1 {
		return "", fmt.Errorf("checksum file format is invalid")
	}

	return parts[0], nil
}



// extractTarGz extracts a .tar.gz archive to a specified location on Windows.
func extractTarGz(tarGzPath, outputPath string) error {
    file, err := os.Open(tarGzPath)
    if err != nil {
        return err
    }
    defer file.Close()

    gzipReader, err := gzip.NewReader(file)
    if err != nil {
        return err
    }
    defer gzipReader.Close()

    tarReader := tar.NewReader(gzipReader)

    for {
        header, err := tarReader.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        targetPath := filepath.Join(outputPath, header.Name)
        switch header.Typeflag {
        case tar.TypeDir:
            if err := os.MkdirAll(targetPath, 0755); err != nil {
                return err
            }
        case tar.TypeReg:
            outFile, err := os.Create(targetPath)
            if err != nil {
                return err
            }
            if _, err := io.Copy(outFile, tarReader); err != nil {
                outFile.Close()
                return err
            }
            outFile.Close()
        }
    }

    return nil
}

// installWasmOpt downloads, verifies the SHA256 hash, and extracts binaryen for a specific version.
func installWasmOpt(version, outputPath string) error {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	baseURL := fmt.Sprintf("https://github.com/WebAssembly/binaryen/releases/download/version_%s/", version)

	var downloadURL string
	switch osName {
	case "linux", "darwin":
		downloadURL = baseURL + fmt.Sprintf("binaryen-version_%s-%s-%s.tar.gz", version, osName, arch)
	case "windows":
		downloadURL = baseURL + fmt.Sprintf("binaryen-version_%s-%s-%s.tar.gz", version, osName, arch)
	default:
		return fmt.Errorf("unsupported OS: %s", osName)
	}

	checksumURL := downloadURL + ".sha256"

	tmpDir := os.TempDir()
	tmpFilePath := filepath.Join(tmpDir, filepath.Base(downloadURL))
	checksumFilePath := tmpFilePath + ".sha256"

	// Download the archive and its checksum
	if err := downloadFile(downloadURL, tmpFilePath); err != nil {
		return fmt.Errorf("failed to download the archive: %w", err)
	}
	if err := downloadFile(checksumURL, checksumFilePath); err != nil {
		return fmt.Errorf("failed to download the checksum: %w", err)
	}

	// Verify the checksum
	expectedHash, err := readChecksum(checksumFilePath)
	if err != nil {
		return fmt.Errorf("failed to read checksum: %w", err)
	}
	if !verifyChecksum(tmpFilePath, expectedHash) {
		return fmt.Errorf("checksum verification failed")
	}

	// Extract the archive based on OS
	if osName == "windows" {
		// Use Go's archive/tar and compress/gzip for Windows
		if err := extractTarGz(tmpFilePath, outputPath); err != nil {
			return fmt.Errorf("failed to extract the archive on Windows: %w", err)
		}
	} else {
		// Use exec.Command for Linux and Darwin
		if err := exec.Command("tar", "-xzf", tmpFilePath, "-C", outputPath).Run(); err != nil {
			return fmt.Errorf("failed to extract the archive on Linux/Darwin: %w", err)
		}
	}

	fmt.Println("binaryen installed and verified successfully")
	return nil
}

// Prompt the user and get a yes/no response
func askUserPermission(question string) bool {
	fmt.Println(question + " (y,n):")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := scanner.Text()
		return strings.ToLower(response) == "y"
	}

	return false
}

// promptUser prompts the user with a question and returns true for "y" and false for "n".
// It repeats the prompt until a valid input is received.
func promptUser(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(question + " (y/n): ")
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading response. Please try again.")
			continue
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response == "y" {
			return true
		} else if response == "n" {
			return false
		} else {
			fmt.Println("Invalid input. Please answer 'y' or 'n'.")
		}
	}
}

// Add the directory to the PATH environment variable
func addToPath(directory string) error {
	var cmd *exec.Cmd

	// Check the operating system
	switch runtime.GOOS {
	case "windows":
		// Windows command to add to PATH for the current user
		cmd = exec.Command("cmd", "/C", "setx", "PATH", fmt.Sprintf(`"%s;%%PATH%%"`, directory))
	case "linux", "darwin":
		// Assume bash shell for Linux and macOS
		profileFile := "$HOME/.bashrc" // You might want to adjust this based on the user's shell
		cmdString := fmt.Sprintf("echo 'export PATH=\"%s:$PATH\"' >> %s", directory, profileFile)
		cmd = exec.Command("bash", "-c", cmdString)
	default:
		return fmt.Errorf("unsupported operating system")
	}

	// Execute the command
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}


func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: binaryen_install <version>")
		os.Exit(1)
	}

	version := os.Args[1] // Get the version from the command line

	// Default outputPath, adjust based on OS
	outputPath := "/usr/local/bin"
	if runtime.GOOS == "windows" {
		outputPath = `C:\Program Files\Binaryen` // Example path for Windows, adjust as needed
	}

	// Call installWasmOpt with the version obtained from the command line
	if err := installWasmOpt(version, outputPath); err != nil {
		fmt.Printf("Error installing wasm-opt: %s\n", err)
		os.Exit(1)
	}

	installationDir := fmt.Sprintf("/usr/local/bin/binaryen_version_%s/bin",version) // Adjust based on actual installation

	// Ask the user for permission to add to PATH using the new prompt function
	if promptUser("Do you want to add Binaryen to your PATH?") {
		if err := addToPath(installationDir); err != nil {
			fmt.Printf("Failed to add Binaryen to PATH. Error: %s\n", err)
			fmt.Println("You may need to add it manually or run this program with elevated permissions.")
		} else {
			fmt.Println("Binaryen was successfully added to your PATH.")
		}
	} else {
		fmt.Println("Binaryen was not added to your PATH.")
	}
}

