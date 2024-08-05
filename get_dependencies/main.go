package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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

	// Get the dependencies of the specified package version
	cmd := exec.Command("npm", "view", fmt.Sprintf("%s@%s", packageName, packageVersion), "dependencies", "--json")
	output, err := cmd.CombinedOutput() // Use CombinedOutput to capture both stdout and stderr
	if err != nil {
		fmt.Printf("Error: %s\n", string(output)) // Print the combined output
		return
	}

	// Handle empty or malformed output
	if len(output) == 0 || string(output) == "null" {
		fmt.Println("No dependencies found for the specified package version.")
		output = []byte("{}") // Ensure output is valid JSON
	}

	// Parse the JSON output
	var dependencies map[string]string
	if err := json.Unmarshal(output, &dependencies); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// If dependencies are not provided, create an empty map
	if len(dependencies) == 0 {
		dependencies = make(map[string]string)
	}

	// Extract the dependency names and sort them
	var depNames []string
	for dep := range dependencies {
		depNames = append(depNames, dep)
	}
	sort.Strings(depNames)

	// Create a map to store the dependencies with their specified versions
	sortedDependencies := make(map[string]string)
	for _, dep := range depNames {
		sortedDependencies[dep] = dependencies[dep]
	}

	// Define the package information structure
	packageInfo := PackageInfo{
		Name:         packageName,
		Version:      packageVersion,
		Dependencies: sortedDependencies,
	}

	// Define file paths
	infoFilePath := getFilePath(packageName, packageVersion)
	latestFilePath := getLatestVersionsFilePath(packageName, packageVersion)

	// Save the package information to a JSON file
	if err := savePackageInfoToFile(infoFilePath, packageInfo); err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Dependencies for %s@%s have been saved to %s\n", packageName, packageVersion, infoFilePath)

	// Get latest versions for dependency ranges and save to a separate file
	if err := saveLatestVersionsToFile(latestFilePath, packageInfo); err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Latest versions for %s@%s dependencies have been saved to %s\n", packageName, packageVersion, latestFilePath)
}

// savePackageInfoToFile saves the package information to a JSON file.
func savePackageInfoToFile(filePath string, packageInfo PackageInfo) error {
	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	// Create JSON encoder and set indentation
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	// Encode and save the package information
	if err := encoder.Encode(packageInfo); err != nil {
		return fmt.Errorf("error writing JSON to file: %w", err)
	}

	return nil
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

	// Define the latest package information structure
	latestPackageInfo := PackageInfo{
		Name:         packageInfo.Name,
		Version:      packageInfo.Version,
		Dependencies: latestVersions,
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	// Create JSON encoder and set indentation
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	// Encode and save the latest package information
	if err := encoder.Encode(latestPackageInfo); err != nil {
		return fmt.Errorf("error writing JSON to file: %w", err)
	}

	return nil
}

func getLatestVersionForRange(packageName, versionRange string) (string, error) {
	// Fetch all versions of the package
	cmd := exec.Command("npm", "view", packageName, "versions", "--json")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse the JSON output
	var versions []string
	if err := json.Unmarshal(output, &versions); err != nil {
		return "", err
	}

	// Find the highest version that matches the range
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
	// Parse the range
	r, err := semver.NewConstraint(versionRange)
	if err != nil {
		return false
	}

	// Parse the version
	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}

	// Check if the version satisfies the range
	return r.Check(v)
}

// compareVersions compares two version strings and returns an integer indicating their order.
func compareVersions(v1, v2 string) int {
	// Implement version comparison logic here.
	return strings.Compare(v1, v2)
}

func parseArguments() (string, string, error) {
	if len(os.Args) < 3 {
		return "", "", fmt.Errorf("Usage: go run main.go <packageName> <version>")
	}

	packageName := os.Args[1]
	packageVersion := os.Args[2]

	return packageName, packageVersion, nil
}

func getFilePath(packageName, packageVersion string) string {
	// Define the file name and path
	fileName := fmt.Sprintf("%s@%s.json", packageName, packageVersion)
	filePath := filepath.Join(config.TestdataPath, fileName)
	return filePath
}

func getLatestVersionsFilePath(packageName, packageVersion string) string {
	// Define the file name and path for latest versions
	fileName := fmt.Sprintf("%s@%s-latest.json", packageName, packageVersion)
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
