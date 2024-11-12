package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/shuffle/shuffle-shared"
)


// VerifyFolder checks a single folder for required files and structure
func VerifyFolder(folderPath string) error {
	// Check that folder exists and is a directory
	info, err := os.Stat(folderPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("invalid folder: %w", err)
	}

	// File paths for api.yaml and app.py
	apiFilePath := fmt.Sprintf("%s/api.yaml", folderPath)
	pythonFilePath := fmt.Sprintf("%s/src/app.py", folderPath)

	// Validate api.yaml contents
	apiData, err := parseAPIYaml(apiFilePath)
	if err != nil {
		return fmt.Errorf("error parsing API YAML in %s: %v", apiFilePath, err)
	}

	// Check for discrepancies in name and version
	if !strings.EqualFold(apiData.Name, folderPath) {
		fmt.Printf("Bad name: %s vs %s\n", folderPath, apiData.Name)
	}
	if apiData.AppVersion != folderPath {
		fmt.Printf("Bad version in %s: expected %s, found %s\n", folderPath, apiData.AppVersion, folderPath)
	}

	// Check unsupported large_image format
	if strings.Contains(apiData.LargeImage, "svg") {
		fmt.Printf("Unsupported large_image format in %s: svg\n", apiFilePath)
	}

	// Validate actions in app.py
	if err := checkActionsInPython(apiData.Actions, pythonFilePath); err != nil {
		fmt.Println(err)
	}

	// Run the Python script and capture output
	if err := runPythonScript(pythonFilePath); err != nil {
		fmt.Printf("Error running Python script %s: %v\n", pythonFilePath, err)
	}

	return nil
}

// parseAPIYaml loads the API data from api.yaml

func parseAPIYaml(apiFilePath string) (*shuffle.WorkflowApp, error) {
	data, err := ioutil.ReadFile(apiFilePath)
	if err != nil {
		return nil, err
	}

	var apiData shuffle.WorkflowApp
	if err := yaml.Unmarshal(data, &apiData); err != nil {
		return nil, fmt.Errorf("YAML parsing error: %w", err)
	}

	return &apiData, nil
}

// checkActionsInPython verifies each action from api.yaml exists in app.py
func checkActionsInPython(actions []shuffle.WorkflowAppAction, pythonFilePath string) error {
	pythonData, err := ioutil.ReadFile(pythonFilePath)
	if err != nil {
		return fmt.Errorf("Error reading Python file %s: %w", pythonFilePath, err)
	}

	missingActions := []string{}
	for _, action := range actions {
		if !strings.Contains(string(pythonData), action.Name) {
			missingActions = append(missingActions, action.Name)
		}
	}

	if len(missingActions) > 0 {
		return fmt.Errorf("Missing actions in %s: %v", pythonFilePath, missingActions)
	}

	return nil
}

// runPythonScript executes a Python script and checks for errors
func runPythonScript(pythonFilePath string) error {
	cmd := exec.Command("python3", pythonFilePath)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check stderr for ModuleNotFoundError specifically
		if strings.Contains(stderr.String(), "ModuleNotFoundError") {
			return nil // Ignore missing modules as per original logic
		}
		return fmt.Errorf("Failed to run script: %v\nStderr: %s", err, stderr.String())
	}

	// Output stdout or stderr
	if stdout.Len() > 0 {
		fmt.Println("Script output:", stdout.String())
	} else if stderr.Len() > 0 {
		fmt.Println("Script error output:", stderr.String())
	}

	return nil
}

/*
func main() {
	// Example usage for a single folder
	folder := "./example_folder"
	if err := VerifyFolder(folder); err != nil {
		fmt.Println("Verification error:", err)
	} else {
		fmt.Println("Folder verification succeeded.")
	}
}
*/
