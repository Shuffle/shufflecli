package main

import (
	"os"
	"fmt"
	"log"
	"io"
	"time"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"net/http"
	"io/ioutil"
	"archive/zip"
	"path/filepath"
	"mime/multipart"

	"github.com/spf13/cobra"

	//"encoding/json"
	//"github.com/shuffle/shuffle-shared"
)


var apikey string
var uploadUrl = "https://shuffler.io"
var orgId = "orgId"
var shuffleCodePath = "./shuffle_code"

func main() {

	shuffleLogo := ``

	rootCmd := &cobra.Command{
		Use:   "shufflecli",
		Short: "Shuffle CLI",
		Long:  "A CLI tool to help with building apps in Shuffle. SHUFFLE_APIKEY, SHUFFLE_URL and SHUFFLE_ORGID environment variables can be used to overwrite the default values.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s\n\nWelcome to the Shuffle CLI! Use -h to see available commands.", shuffleLogo)
		},
	}

	if os.Getenv("SHUFFLE_APIKEY") == "" && os.Getenv("SHUFFLE_AUTHORIZATION") == "" {
		fmt.Println("Please set the SHUFFLE_APIKEY and SHUFFLE_AUTHORIZATION environment variables to help with upload/download.")
	}

	if len(os.Getenv("SHUFFLE_APIKEY")) > 0 {
		apikey = os.Getenv("SHUFFLE_APIKEY")
	} else if len(os.Getenv("SHUFFLE_AUTHORIZATION")) > 0 {
		apikey = os.Getenv("SHUFFLE_AUTHORIZATION")
	}

	if len(os.Getenv("SHUFFLE_URL")) > 0 {
		uploadUrl = os.Getenv("SHUFFLE_URL")
	}

	if len(os.Getenv("SHUFFLE_ORGID")) > 0 {
		orgId = os.Getenv("SHUFFLE_ORGID")
	}

	if len(os.Getenv("SHUFFLE_CODEPATH")) > 0 {
		shuffleCodePath = os.Getenv("SHUFFLE_CODEPATH")
	}

	// Adding commands to root
	//rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(appCmd)
	rootCmd.AddCommand(devCmd)
	//rootCmd.AddCommand(mathCmd)

	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Example command: Display version information
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ShuffleCLI v0.0.1")
	},
}

func TestApp(cmd *cobra.Command, args []string) {
	log.Printf("[DEBUG] Testing app config: %s", args)

	if len(args) <= 0 {
		log.Printf("[ERROR] No directory provided. Use the absolute path to the app directory.")
		return
	}

	err := runUploadValidation(args)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			if strings.Contains(err.Error(), "api.yaml") {
				log.Printf("[ERROR] Can't find api.yaml file in '%s'. Make sure to point into a VERSION of the app, containing the 'src' folder.", args[0])
			} else if strings.Contains(err.Error(), "app.py") {
				log.Printf("[ERROR] Can't find app.py file in '%s'. Make sure to point into a VERSION of the app, containing the 'src' folder.", args[0])
			} else {
				log.Printf("[ERROR] Can't find app folder '%s'. Use the absolute path.", args[0])
			}

			return
		}

		log.Printf("[ERROR] App validation issue: %s", err)
		return
	}

	log.Printf("[INFO] App validated successfully. Upload it with command: \n'shufflecli app upload %s'", args[0])
}

// Example command: Greet the user
var testApp = &cobra.Command{
	Use:   "test",
	Short: "Tests an app",
	Run: func(cmd *cobra.Command, args []string) {
		TestApp(cmd, args)
	},
}

var runApp = &cobra.Command{
	Use:   "run",
	Short: "Tests and app (synonym)",
	Run: func(cmd *cobra.Command, args []string) {
		TestApp(cmd, args)
	},
}

func validatePythonfile(filepath string) error {
	cmd := exec.Command("python3", "-m", "pip", "install", "shuffle_sdk", "--upgrade", "--break-system-packages")
	log.Printf("[DEBUG] Ensuring shuffle-sdk is installed for testing")

	var stdoutBuffer, stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	if err := cmd.Run(); err != nil {
		log.Printf("[ERROR] Problem installing SDK: %s", err)

		stdout := stdoutBuffer.String()
		if len(stdout) > 0 {
			log.Printf("\n\nOutput: %s\n\n", stdout)
		}

		stderr := stderrBuffer.String()
		if len(stderr) > 0 {
			log.Printf("\n\nError: %s\n\n", stderr)
		}
		return err
	}

	stdoutBuffer.Reset()
	stderrBuffer.Reset()

	// Run requirements install 

	tmpFilepath := filepath
	if strings.HasSuffix(filepath, "/src/app.py") {
		tmpFilepath = filepath[:len(filepath)-len("/src/app.py")]
	}

	cmd = exec.Command("python3", "-m", "pip", "install", "-r", fmt.Sprintf("%s/requirements.txt", tmpFilepath), "--break-system-packages")
	log.Printf("[DEBUG] Installing requirements for testing")
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	if err := cmd.Run(); err != nil {
		log.Printf("[ERROR] Problem installing from requirements file: %s", err)

		stdout := stdoutBuffer.String()
		if len(stdout) > 0 {
			log.Printf("\n\nOutput: %s\n\n", stdout)
		}

		stderr := stderrBuffer.String()
		if len(stderr) > 0 {
			log.Printf("\n\nError: %s\n\n", stderr)
		}
		return err
	}

	// Make a copy of the app.py file and run it
	copyFilepath := "/tmp/shuffle_app.py"
	if len(os.Getenv("TESTDIR")) > 0 { 
		copyFilepath = fmt.Sprintf("%s/shuffle_app.py", os.Getenv("TESTDIR"))
	}

	newFile, err := os.Create(copyFilepath)
	if err != nil {
		log.Printf("[ERROR] Problem creating copy of python file: %s", err)
		return err
	}

	defer newFile.Close()
	original, err := os.Open(filepath)
	if err != nil {
		log.Printf("[ERROR] Problem opening python file: %s", err)
		return err
	}

	// Read the content of outFile and change it
	filedata, err := ioutil.ReadAll(original)
	if err != nil {
		log.Printf("[ERROR] Problem reading original app.py file: %s", err)
		return err
	}

	filedata = []byte(strings.Replace(string(filedata), "from walkoff_app_sdk.app_base", "from shuffle_sdk", -1))

	// Write it back to the new file
	_, err = newFile.Write(filedata)
	if err != nil {
		log.Printf("[ERROR] Problem writing to new app.py file: %s", err)
		return err
	}

	log.Printf("[DEBUG] Copying app.py file to %s to make edits for the test", copyFilepath)

	//pythonCommand := fmt.Sprintf("python3 %s", filepath)

	// Any way we can INJECT the shuffle/walkoff API into the python file?

	// Run the python file as a test
	// Clear buffers

	pythonCommand := fmt.Sprintf("python3 %s", copyFilepath)

	timeout := 3 * time.Second
	log.Printf("[DEBUG] Validating python file by running '%s' for up to %d seconds.", pythonCommand, int(timeout)/1000000000)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // Ensure resources are released

	// Run for maximum 5 seconds
	//cmd = exec.Command("python3", copyFilepath)
	cmd = exec.CommandContext(ctx, "python3", copyFilepath)
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Println("Command timed out")
	}

	if err != nil {
		if strings.Contains(err.Error(), "signal: killed") {
			err = nil
		}

		if err != nil {
			log.Printf("[ERROR] Local run of python file: %s", err)
		}

		stdout := stdoutBuffer.String()
		if len(stdout) > 0 {
			log.Printf("\n\n===== Python run (stdout) ===== \n")

			for _, line := range strings.Split(stdout, "\n") {
				if strings.Contains(strings.ToLower(line), "traceback") && !strings.Contains(strings.ToLower(line), "Bad resp") {
					log.Printf("[ERROR] Python run Error: %s", line)
				} else if strings.Contains(strings.ToLower(line), "already satisfied") {
					continue
				} else {
					fmt.Println(line)
				}
			}
		}

		stderr := stderrBuffer.String()
		if len(stderr) > 0 {
			log.Printf("\n\n===== Python run (stderr) ===== \n")

			for _, line := range strings.Split(stdout, "\n") {
				if strings.Contains(strings.ToLower(line), "traceback") {
					log.Printf("[ERROR] Python run Error: %s", line)
				} else if strings.Contains(strings.ToLower(line), "already satisfied") || strings.Contains(strings.ToLower(line), "[ERROR]") || strings.Contains(strings.ToLower(line), "[WARNING]") || strings.Contains(strings.ToLower(line), "[INFO]") || strings.Contains(strings.ToLower(line), "[DEBUG]") {
					continue
				} else {
					fmt.Println(line)
				}
			}

		}

		return err
	}

	// Remove the copy
	/*
	err = os.Remove(copyFilepath)
	if err != nil {
		log.Printf("[ERROR] Problem removing copy of python file '%s': %s", copyFilepath, err)
	}
	*/

	log.Printf("[INFO] Python file ran successfully\n")

	return nil
}

func validateAppFilepath(filepath string) error {
	fileStat, err := os.Stat(filepath) 
	if err != nil {
		log.Printf("Directory '%s' does not exist.", filepath)
		return err
	}

	_ = fileStat
	yamlFile := fmt.Sprintf("%s/api.yaml", filepath)
	pyFile := fmt.Sprintf("%s/src/app.py", filepath)
	requirementsFile := fmt.Sprintf("%s/requirements.txt", filepath)

	// Check if the files exist
	if _, err := os.Stat(yamlFile); os.IsNotExist(err) {
		log.Printf("[ERROR] YAML file '%s' does not exist in directory %s.", yamlFile, filepath)
		return err
	}

	if _, err := os.Stat(pyFile); os.IsNotExist(err) {
		log.Printf("[ERROR] Python file '%s' does not exist in %s.", pyFile, filepath)
		return err
	}

	if _, err := os.Stat(requirementsFile); os.IsNotExist(err) {
		log.Printf("[ERROR] Requirements file '%s' does not exist in %s.", requirementsFile, filepath)
		return err
	}

	log.Printf("[INFO] All relevant files exist.")
	return nil
}

func runUploadValidation(args []string) error {
	err := validateAppFilepath(args[0])
	if err != nil {
		log.Printf("[ERROR] Failed validating app directory: %s", err)
		return err
	}

	errors, err := VerifyFolder(args[0])
	if err != nil {
		log.Printf("[ERROR] Problem verifying folder %s: %s", args[0], err)
		return err
	}

	if len(errors) > 0 {
		log.Printf("[ERROR] App validation failed. Please fix the following issues: '%s'. Read the above logs to learn about these.", strings.Join(errors, ", "))
		return fmt.Errorf("Validation failed because of %s", strings.Join(errors, ", "))
	}

	pyFile := fmt.Sprintf("%s/src/app.py", args[0])
	err = validatePythonfile(pyFile) 
	if err != nil {
		log.Printf("[ERROR] Problem validating python file: %s", err)
		return err
	}

	log.Printf("[INFO] Zip + Uploading app from directory: %s", args[0])
	return nil
}

func ZipFiles(filename string, files []string) error {
	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {
		zipfile, err := os.Open(file)
		if err != nil {
			return err
		}
		defer zipfile.Close()

		// Get the file information
		info, err := zipfile.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Using FileInfoHeader() above only uses the basename of the file. If we want
		// to preserve the folder structure we can overwrite this with the full path.
		filesplit := strings.Split(file, "/")
		if len(filesplit) > 1 {
			header.Name = filesplit[len(filesplit)-1]
		} else {
			header.Name = file
		}

		// Change to deflate to gain better compression
		// see http://golang.org/pkg/archive/zip/#pkg-constants
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err = io.Copy(writer, zipfile); err != nil {
			return err
		}
	}

	return nil
}

func UploadAppFromRepo(folderpath string) error {
	log.Printf("[DEBUG] Uploading app from %#v: ", folderpath)


	// Walk the path and add 
	allFiles := []string{}
	err := filepath.Walk(folderpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			allFiles = append(allFiles, path)
		}

		return nil
	})

	if err != nil {
		log.Printf("[ERROR] Problem walking path: %s", err)
	}

	zipLocation := fmt.Sprintf("%s/upload.zip", folderpath)
	err = ZipFiles(zipLocation, allFiles)
	if err != nil {
		log.Printf("[ERROR] Problem zipping files: %s", err)
		return err
	}

	newUrl := fmt.Sprintf("%s/api/v1/apps/upload", uploadUrl)
	log.Printf("\n\n[INFO] Zipped files to %s. Starting upload to %s. This may take a while, as validation will take place on cloud.", zipLocation, newUrl)

	// Add file to request
	file, err := os.Open(zipLocation)
	if err != nil {
		log.Printf("[ERROR] Problem opening file: %s", err)
		return err
	}

	defer file.Close()
	body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    // Add the file to the form with the field name "shuffle_file"
    part, err := writer.CreateFormFile("shuffle_file", filepath.Base(zipLocation))
    if err != nil {
		return err
    }

    // Copy the file into the form
    _, err = io.Copy(part, file)
    if err != nil {
		return err
    }

    // Close the multipart writer to finalize the form
    err = writer.Close()
    if err != nil {
		return err
    }

	client := &http.Client{}
	req, err := http.NewRequest(
		"POST",
		newUrl,
		body,
	)

	if err != nil {
		log.Printf("[ERROR] Problem creating request: %s", err)
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apikey))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Upload the file
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Problem uploading file: %s", err)
		return err
	}

	outputBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Problem reading response body: %s", err)
		return err
	}

	/*
	mappedValue := shuffle.RequestResponse{}
	unmarshalErr := json.Unmarshal(outputBody, &mappedValue)
	if unmarshalErr != nil {
		log.Printf("[ERROR] Problem unmarshalling response: %s", unmarshalErr)
		//return unmarshalErr
	} else {
		outputBody = []byte(fmt.Sprintf("Raw output: %s", mappedValue.Details))
	}

	if len(mappedValue.Details) > 0 {
		log.Printf("[INFO] Upload Details: %s", mappedValue.Details)
	}
	*/

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad status: %s. Raw: %s", resp.Status, string(outputBody))
	}

	log.Printf("[INFO] File uploaded successfully: %s", resp.Status)

	return nil
}

var runParameter = &cobra.Command{
	Use:  "run",
	Short: "Run a python script as if it is in the Shuffle UI",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) <= 0 {
			log.Println("[ERROR] No URL provided. Use the URL from the Shuffle UI.")
			return
		}

		if len(apikey) <= 0 {
			fmt.Println("Please set the SHUFFLE_APIKEY or SHUFFLE_AUTHORIZATION environment variables to help with upload/download.")
			os.Exit(1)
		}

		log.Printf("[DEBUG] Running command: %s", args)
		if !strings.Contains(args[0], "http") {
			log.Printf("[ERROR] Go to the Shuffle UI and click the 'Expand code editor' button next to ANY parameter. Paste the URL here. Should contain action_id, action_name and field.")
			os.Exit(1)
		}

		// Parse out parameters from: http://localhost:3002/workflows/5fa39e8a-b70c-4343-93ff-d49cfc12148e?action_id=643059ab-2278-4c87-9168-a61ef73f4ed3&field=call&action_name=repeat_back_to_me
		// Get the workflow ID, action_id, action_name and field

		urlsplit := strings.Split(args[0], "workflows/")
		if len(urlsplit) <= 1 {
			log.Printf("[ERROR] URL doesn't contain any workflows. Please paste the URL from the Shuffle UI, or ensure it contains the action_id, action_name and fields.")
			os.Exit(1)
		}

		// Split the URL by ?
		urlParts := strings.Split(urlsplit[1], "?")
		if len(urlParts) <= 1 {
			log.Printf("[ERROR] URL doesn't contain any parameters. Please paste the URL from the Shuffle UI, or ensure it contains the action_id, action_name and fields.")
			os.Exit(1)
		}

		workflowId := urlParts[0]
		if len(workflowId) <= 0 {
			log.Printf("[ERROR] Workflow ID not found. Please paste the URL from the Shuffle UI, or ensure it contains the action_id, action_name and fields.")
			os.Exit(1)
		}

		log.Printf("[DEBUG] Workflow ID: %s", workflowId)
		workflow, err := GetWorkflow(workflowId)
		if err != nil {
			log.Printf("[ERROR] Problem getting workflow: %s", err)
			os.Exit(1)
		}

		// Split the parameters by &
		parameters := strings.Split(urlParts[1], "&")
		if len(parameters) <= 1 {
			log.Printf("[ERROR] URL doesn't contain any parameters. Please paste the URL from the Shuffle UI, or ensure it contains the action_id, action_name and fields.")
			os.Exit(1)
		}


		actionId := ""
		actionName := ""
		field := ""
		for _, param := range parameters {
			stringParts := strings.Split(param, "=")
			if len(stringParts) <= 1 {
				log.Printf("[ERROR] Parameter doesn't contain a value, and is '%s'. Please paste the URL from the Shuffle UI, or ensure it contains the action_id, action_name and fields.", param)
				continue
			}

			if stringParts[0] == "action_id" {
				actionId = stringParts[1]
			}

			if stringParts[0] == "action_name" {
				actionName = stringParts[1]
			}

			if stringParts[0] == "field" {
				field = stringParts[1]
			}
		}

		if field != "code" || actionName != "execute_python" {
			log.Printf("[ERROR] This command only works for the 'execute_python' action and 'code' field so far. Got: %s and %s", actionName, field)
			os.Exit(1)
		}

		log.Printf("Action ID: %s, Action Name: %s, Field: %s", actionId, actionName, field)

		foundActionIndex := -1
		foundParamIndex := -1

		for actionIndex, action := range workflow.Actions {
			if action.ID != actionId {
				continue
			}

			// Found the action
			log.Printf("[INFO] Found action: %s", action.Name)

			foundActionIndex = actionIndex

			for _, param := range action.Parameters {
				if param.Name != field {
					continue
				}

				foundParamIndex = actionIndex

				// Found the parameter
				log.Printf("[INFO] Found parameter: %s", param.Name)

				// Should download the value and put it in a local file
			}
		}

		if foundActionIndex == -1 {
			log.Printf("[ERROR] Action ID not found in workflow.")
			os.Exit(1)
		}

		if foundParamIndex == -1 {
			log.Printf("[ERROR] Parameter not found in action.")
			os.Exit(1)
		}

		log.Printf("[INFO] Running action: %s, parameter: %s", actionName, field)
		// Check if folder ./shuffle_code exists, otherwise make it
		if _, err := os.Stat(shuffleCodePath); os.IsNotExist(err) {
			os.Mkdir(shuffleCodePath, os.ModePerm)
		}

		filepath := fmt.Sprintf("./shuffle_code/%s_%s.py", field, actionId)

		// Write the code to the file
		actionCode := workflow.Actions[foundActionIndex].Parameters[foundParamIndex].Value

		err = ioutil.WriteFile(filepath, []byte(actionCode), 0644)
		if err != nil {
			log.Printf("[ERROR] Problem writing code to file: %s", err)
			os.Exit(1)
		}

		dockerCommand := fmt.Sprintf("docker run frikky/shuffle:shuffle_tools-1.2.0 TBD")
		log.Printf("[INFO] Start editing the file here below.. Saving it will upload it automatically. Workflow Revisions will keep track of old versions, so don't worry too much. \n\nPATH: %s\n\nTest using the following command locally (requires Docker running): \n%s\n", filepath, dockerCommand)

		// FIXME: Start listener for changes
		for {
			// Check if the file has changed
			// If so, upload it
			// If not, wait 5 seconds

			// Check if the file has changed
			newCode := ""
			newCodeBytes, err := ioutil.ReadFile(filepath)
			if err != nil {
				log.Printf("[ERROR] Problem reading file: %s", err)
				break
			}

			newCode = string(newCodeBytes)
			if newCode == actionCode {
				time.Sleep(1 * time.Second)
				continue
			}


			actionCode = newCode
			workflow.Actions[foundActionIndex].Parameters[foundParamIndex].Value = actionCode
			go UploadWorkflow(workflow)

			log.Printf("[INFO] Code changed and uploading.", dockerCommand)

			time.Sleep(1 * time.Second)
		}
	},
}

var uploadApp = &cobra.Command{
	Use:   "upload",
	Short: "Uploads and app from a directory containing the api.yaml",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) <= 0 {
			log.Println("[ERROR] No directory provided. Use the absolute path to the app directory.")
			return
		}

		if len(apikey) <= 0 {
			fmt.Println("Please set the SHUFFLE_APIKEY or SHUFFLE_AUTHORIZATION environment variables to help with upload/download.")
			os.Exit(1)
		}

		// Look for if there is a filepath or not, which contains an api.yaml file AND a src/app.py file
		if len(args) <= 0 {
			args = append(args, ".")
			log.Println("[DEBUG] No directory provided. Using current directory.")
		}

		err := runUploadValidation(args)
		if err != nil {
			if strings.Contains(err.Error(), "no such file") {
				if strings.Contains(err.Error(), "api.yaml") {
					log.Printf("[ERROR] Can't find api.yaml file in '%s'. Make sure to point into a VERSION of the app, containing the 'src' folder.", args[0])
				} else if strings.Contains(err.Error(), "app.py") {
					log.Printf("[ERROR] Can't find app.py file in '%s'. Make sure to point into a VERSION of the app, containing the 'src' folder.", args[0])
				} else {
					log.Printf("[ERROR] Can't find app folder '%s'. Use the absolute path.", args[0])
				}

				return
			}

			log.Printf("[ERROR] App validation issue: %s", err)
			//return
		}

		// Get user input for whether to continue or not with Y/n
		input := "Y"	
		fmt.Print("\n\nContinue with upload? [Y/n]: ")
		fmt.Scanln(&input)

		log.Printf("INPUT: %#v", input)
		if strings.ToUpper(input) != "Y" {
			log.Println("[INFO] Aborting upload.")
			return
		}

		// Upload the app
		err = UploadAppFromRepo(args[0])
		if err != nil {
			log.Printf("[ERROR] Problem uploading app: %s", err)
			return
		}

		log.Println("[INFO] App uploaded successfully.")
	},
}

// Example command with subcommands: Math operations
var appCmd = &cobra.Command{
	Use:   "app",
	Short: "App related commands",
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Development related commands",
}

func init() {
	// Register subcommands to the math command
	appCmd.AddCommand(uploadApp)
	appCmd.AddCommand(testApp)

	devCmd.AddCommand(runParameter)
}

