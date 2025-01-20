package main

import (
	"log"
	"fmt"
	"bytes"
	"strings"
	"net/http"
	"io/ioutil"
	"encoding/json"

	"github.com/shuffle/shuffle-shared"
)

func GetWorkflow(workflowId string) (shuffle.Workflow, error) {
	workflow := shuffle.Workflow{}
	baseUrl := uploadUrl 
	url := fmt.Sprintf("%s/api/v1/workflows/%s", baseUrl, workflowId)

	client := &http.Client{}
	req, err := http.NewRequest(
		"GET", 
		url,
		nil,
	)

	if err != nil {
		log.Printf("[ERROR] Failed to create request: %v\n", err)
		return workflow, err
	}

	if strings.HasPrefix(apikey, "Bearer ") {
		req.Header.Add("Authorization", apikey)
	} else {
		req.Header.Add("Authorization", "Bearer "+apikey)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Failed to send request: %v\n", err)
		return workflow, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("[ERROR] Failed to get workflow: %v\n", resp.Status)
		return workflow, fmt.Errorf("Failed to get workflow: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read response: %v\n", err)
		return workflow, err
	}

	if err := json.Unmarshal(body, &workflow); err != nil {
		log.Printf("[ERROR] Failed to unmarshal response: %v\n", err)
		return workflow, err
	}

	return workflow, nil
}

func UploadWorkflow(workflow shuffle.Workflow) error {
	baseUrl := uploadUrl 
	url := fmt.Sprintf("%s/api/v1/workflows/%s", baseUrl, workflow.ID)

	payload, err := json.Marshal(workflow)	
	if err != nil {
		log.Printf("[ERROR] Failed to marshal workflow: %v\n", err)
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest(
		"PUT", 
		url,
		bytes.NewBuffer(payload),
	)

	if err != nil {
		log.Printf("[ERROR] Failed to create request: %v\n", err)
		return err
	}

	if strings.HasPrefix(apikey, "Bearer ") {
		req.Header.Add("Authorization", apikey)
	} else {
		req.Header.Add("Authorization", "Bearer "+apikey)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Failed to send request: %v\n", err)
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("[ERROR] Failed to upload workflow: %v\n", resp.Status)
		return fmt.Errorf("Failed to get workflow: %v", resp.Status)
	}

	// Check if success: true
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read response: %v\n", err)
		return err
	}

	var response shuffle.ResultChecker
	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("[ERROR] Failed to unmarshal response: %v\n", err)
		return err
	}

	if !response.Success {
		log.Printf("[ERROR] Failed to upload workflow: %#v\n", string(body))
		return fmt.Errorf("Failed to upload workflow: %#v", string(body))
	}

	return nil
}
