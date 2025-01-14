package main

import (
	"log"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/shuffle/shuffle-shared"
)


// VerifyFolder checks a single folder for required files and structure
func VerifyFolder(folderPath string) ([]string, error) {
	fmt.Println()

	if strings.HasSuffix(folderPath, "/") {	
		folderPath = folderPath[:len(folderPath)-1]
	}

	errors := []string{}

	// Check that folder exists and is a directory
	info, err := os.Stat(folderPath)
	if err != nil || !info.IsDir() {
		return errors, fmt.Errorf("invalid folder: %w", err)
	}

	// File paths for api.yaml and app.py
	apiFilePath := fmt.Sprintf("%s/api.yaml", folderPath)
	pythonFilePath := fmt.Sprintf("%s/src/app.py", folderPath)

	// Validate api.yaml contents
	apiData, err := parseAPIYaml(apiFilePath)
	if err != nil {
		return errors, fmt.Errorf("error parsing API YAML in %s: %v", apiFilePath, err)
	}

	// Check for discrepancies in name and version
	if !strings.EqualFold(apiData.Name, folderPath) {
		log.Printf("[ERROR] Bad name: '%s' vs '%s' in api.yaml\n", folderPath, apiData.Name)
		errors = append(errors, "appname")
	}

	folderVersion := strings.Split(folderPath, "-")[len(strings.Split(folderPath, "-"))-1]
	if apiData.AppVersion != folderVersion {
		log.Printf("[ERROR] Bad version in %s: expected %s, found %s\n", folderPath, apiData.AppVersion, folderVersion)
		errors = append(errors, "folder version")
	}

	// Check unsupported large_image format
	if strings.Contains(apiData.LargeImage, "svg") {
		log.Printf("[ERROR] Unsupported large_image format in %s: svg\n", apiFilePath)
		errors = append(errors, "image")
	}

	// Validate actions in app.py
	if err := checkActionsInPython(apiData.Actions, pythonFilePath); err != nil {
		log.Printf("[ERROR] Problem with python check: %s", err)
		errors = append(errors, "actions")
	}

	fmt.Println()

	return errors, nil
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
