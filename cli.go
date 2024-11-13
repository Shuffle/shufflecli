package main

import (
	"os"
	"fmt"
	"log"
	"io"
	"bytes"
	"os/exec"
	"strings"
	"io/ioutil"
	"net/http"
	"archive/zip"
	"mime/multipart"

	"path/filepath"
	"github.com/spf13/cobra"
)


var apikey string
var uploadUrl = "https://shuffler.io"
var orgId = "orgId"

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

	// Adding commands to root
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(appCmd)
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
	log.Printf("Testing app: ", args)
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

	pythonCommand := fmt.Sprintf("python3 %s", filepath)
	log.Printf("[DEBUG] Validating python file by running '%s'", pythonCommand)

	// Any way we can INJECT the shuffle/walkoff API into the python file?

	// Run the python file as a test
	// Clear buffers

	cmd = exec.Command("python3", filepath)
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err := cmd.Run()
	if err != nil {
		log.Printf("[ERROR] Local run of python file: %s", err)

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

	log.Printf("[INFO] Python file ran successfully")

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

	log.Printf("[INFO] All files exist. Starting upload to %s", uploadUrl)
	return nil
}

func runUploadValidation(args []string) error {
	err := validateAppFilepath(args[0])
	if err != nil {
		log.Printf("[ERROR] Failed validating app directory: %s", err)
		return err
	}

	pyFile := fmt.Sprintf("%s/src/app.py", args[0])
	err = validatePythonfile(pyFile) 
	if err != nil {
		log.Printf("[ERROR] Problem validating python file: %s", err)
		return err
	}

	err = VerifyFolder(args[0])
	if err != nil {
		log.Printf("[ERROR] Problem verifying folder %s: %s", args[0], err)
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad status: %s. Raw: %s", resp.Status, string(outputBody))
	}

	log.Printf("[INFO] File uploaded successfully: %s", resp.Status)

	return nil
}

var uploadApp = &cobra.Command{
	Use:   "upload",
	Short: "Uploads and app from a directory containing the api.yaml",
	Run: func(cmd *cobra.Command, args []string) {
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
			log.Printf("[ERROR] Problem with validation: %s", err)
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

func init() {
	// Register subcommands to the math command
	appCmd.AddCommand(uploadApp)

	//appCmd.AddCommand(testApp)
	//appCmd.AddCommand(runApp)
}

