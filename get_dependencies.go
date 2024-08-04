package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

type PackageInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

func main() {
	packageName, packageVersion, err := parseArguments()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Package Name: %s\n", packageName)
	fmt.Printf("Package Version: %s\n", packageVersion)

	// Get the dependencies of the specified package version
	cmd := exec.Command("npm", "view", fmt.Sprintf("%s@%s", packageName, packageVersion), "dependencies", "--json")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Parse the JSON output
	var dependencies map[string]string
	if err := json.Unmarshal(output, &dependencies); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
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

	filePath := getFilePath(packageName, packageVersion)

	if err := savePackageInfoToFile(filePath, packageInfo); err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Dependencies for %s@%s have been saved to %s\n", packageName, packageVersion, filePath)
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

func parseArguments() (string, string, error) {
	if len(os.Args) < 3 {
		return "", "", fmt.Errorf("Usage: go run main.go <packageName> <version>")
	}

	packageName := os.Args[1]
	packageVersion := os.Args[2]

	return packageName, packageVersion, nil
}

func getFilePath(packageName string, packageVersion string) string {

	// Define the file name and path
	fileName := fmt.Sprintf("%s@%s.json", packageName, packageVersion)
	filePath := filepath.Join("testdata", fileName)

	return filePath
}
