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
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <packageName> <version>")
		return
	}

	packageName := os.Args[1]
	packageVersion := os.Args[2]

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

	// Define the file name and path
	fileName := fmt.Sprintf("%s@%s.json", packageName, packageVersion)
	filePath := filepath.Join("testdata", fileName)

	// Save the package information to a JSON file
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(packageInfo); err != nil {
		fmt.Println("Error writing JSON to file:", err)
		return
	}

	fmt.Printf("Dependencies for %s@%s have been saved to %s\n", packageName, packageVersion, filePath)
}
