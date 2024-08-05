package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
)

type Config struct {
	TestdataPath string `json:"testdata_path"`
}

type PackageInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

var config Config

func main() {

	// Load configuration
	if err := loadConfig("../config.json"); err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	packageName, packageVersion, err := parseArguments()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Package Name: %s\n", packageName)
	fmt.Printf("Package Version: %s\n", packageVersion)

	// Get all dependencies (direct and transitive) of the specified package version
	allDependencies, err := getAllDependencies(packageName, packageVersion)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Define the package information structure
	packageInfo := PackageInfo{
		Name:         packageName,
		Version:      packageVersion,
		Dependencies: allDependencies,
	}

	// Define the file path for the latest versions of all dependencies
	latestFilePath := getLatestVersionsFilePath(packageName, packageVersion)

	// Save latest versions of all dependencies to a JSON file
	if err := saveLatestVersionsToFile(latestFilePath, packageInfo); err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Latest versions for all dependencies have been saved to %s\n", latestFilePath)
}

// getAllDependencies recursively fetches all dependencies for a given package version.
func getAllDependencies(packageName, packageVersion string) (map[string]string, error) {
	dependencies := make(map[string]string)
	toProcess := []string{fmt.Sprintf("%s@%s", packageName, packageVersion)}

	seen := make(map[string]bool)

	for len(toProcess) > 0 {
		current := toProcess[0]
		toProcess = toProcess[1:]

		if seen[current] {
			continue
		}
		seen[current] = true

		// Get the dependencies for the current package
		cmd := exec.Command("npm", "view", current, "dependencies", "--json")
		//output, err := cmd.Output()
		output, err := cmd.CombinedOutput() // Use CombinedOutput to capture both stdout and stderr
		if err != nil {
			fmt.Printf("Error: %s\n", string(output)) // Print the combined output
			return nil, err
		}

		if len(output) == 0 || string(output) == "null" {
			output = []byte("{}")
		}

		var deps map[string]string
		if err := json.Unmarshal(output, &deps); err != nil {
			return nil, err
		}

		for dep, versionRange := range deps {
			dependencies[dep] = versionRange
			if !seen[dep] {
				toProcess = append(toProcess, dep)
			}
		}
	}

	return dependencies, nil
}

// saveLatestVersionsToFile saves the latest matching versions of dependencies to a JSON file.
func saveLatestVersionsToFile(filePath string, packageInfo PackageInfo) error {
	latestVersions := make(map[string]string)
	for dep, versionRange := range packageInfo.Dependencies {
		latestVersion, err := getLatestVersionForRange(dep, versionRange)
		if err != nil {
			return fmt.Errorf("error fetching latest version for %s: %w", dep, err)
		}
		latestVersions[dep] = latestVersion
	}

	// Create a map for the output JSON structure
	latestPackageInfo := PackageInfo{
		Name:         packageInfo.Name,
		Version:      packageInfo.Version,
		Dependencies: latestVersions,
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(latestPackageInfo); err != nil {
		return fmt.Errorf("error writing JSON to file: %w", err)
	}

	return nil
}

func getLatestVersionForRange(packageName, versionRange string) (string, error) {
	cmd := exec.Command("npm", "view", packageName, "versions", "--json")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var versions []string
	if err := json.Unmarshal(output, &versions); err != nil {
		return "", err
	}

	latestVersion := ""
	for _, version := range versions {
		if isVersionInRange(version, versionRange) {
			if latestVersion == "" || compareVersions(version, latestVersion) > 0 {
				latestVersion = version
			}
		}
	}

	if latestVersion == "" {
		return "", fmt.Errorf("no version found for %s in range %s", packageName, versionRange)
	}

	return latestVersion, nil
}

func isVersionInRange(version, versionRange string) bool {
	r, err := semver.NewConstraint(versionRange)
	if err != nil {
		return false
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}

	return r.Check(v)
}

func compareVersions(v1, v2 string) int {
	v1Ver, err := semver.NewVersion(v1)
	if err != nil {
		return 0
	}
	v2Ver, err := semver.NewVersion(v2)
	if err != nil {
		return 0
	}
	return v1Ver.Compare(v2Ver)
}

func parseArguments() (string, string, error) {
	if len(os.Args) < 3 {
		return "", "", fmt.Errorf("Usage: go run main.go <packageName> <version>")
	}

	packageName := os.Args[1]
	packageVersion := os.Args[2]

	return packageName, packageVersion, nil
}

func getLatestVersionsFilePath(packageName, packageVersion string) string {
	fileName := fmt.Sprintf("%s@%s-latest-all.json", packageName, packageVersion)
	filePath := filepath.Join(config.TestdataPath, fileName)
	return filePath
}

// loadConfig loads configuration from a JSON file.
func loadConfig(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening config file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("error decoding config file: %w", err)
	}

	return nil
}
