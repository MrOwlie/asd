// asd.go
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type Command struct {
	Name      string `yaml:"name"`
	Extension string `yaml:"extension"`
}

type Category struct {
	Name          string     `yaml:"name"`
	Path          string     `yaml:"path"`
	Commands      []Command  `yaml:"commands"`
	Subcategories []Category `yaml:"subcategories"`
}

type Config struct {
	Categories []Category `yaml:"categories"`
}

type GPT4Request struct {
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens"`
}

type GPT4Response struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

var newCategoryCmd = &cobra.Command{
	Use:   "new-category [name] [parentCategoryName]",
	Short: "Creates a new category",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		categoryName := args[0]
		parentCategoryName := ""
		parentCategoryPath := "."
		if len(args) > 1 {
			parentCategoryName = args[1]
			// Find the parent category path (replace with your actual function)
			var err error
			// Read existing YAML
			data, err := ioutil.ReadFile("commands.yaml")
			if err != nil {
				fmt.Printf("Could not read commands.yaml: %s\n", err)
				return
			}

			var config Config
			err = yaml.Unmarshal(data, &config)
			if err != nil {
				fmt.Printf("Could not unmarshal commands.yaml: %s\n", err)
				return
			}
			parentCategoryPath, err = findCategoryPath(parentCategoryName, config.Categories)
			if err != nil {
				fmt.Printf("Parent category not found: %s\n", err)
				return
			}
		}

		err := updateYAMLWithNewCategory(categoryName, parentCategoryName)
		if err != nil {
			fmt.Printf("Failed to update YAML: %s\n", err)
			return
		}

		err = createCategoryFolder(categoryName, parentCategoryPath)
		if err != nil {
			fmt.Printf("Failed to create folder: %s\n", err)
			return
		}

		fmt.Printf("Added new category: %s\n", categoryName)
	},
}

var newGoCommandCmd = &cobra.Command{
	Use:   "new-go-command [commandName] [parentCategoryName]",
	Short: "Creates a new Go command under a category or subcategory",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		commandName := args[0]
		parentCategoryName := args[1]

		// Read existing YAML
		data, err := ioutil.ReadFile("commands.yaml")
		if err != nil {
			panic(err)
		}

		var config Config
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			panic(err)
		}

		parentCategory, err := findParentCategory(parentCategoryName, config.Categories)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}

		err = addNewGoCommandToCategory(commandName, parentCategory, &config)
		if err != nil {
			fmt.Printf("Failed to add new Go command: %s\n", err)
			return
		}

		fmt.Printf("Added new Go command: %s under category: %s\n", commandName, parentCategoryName)
	},
}

var scriptContent string

var newLinuxCommandCmd = &cobra.Command{
	Use:   "new-linux-command [name] [category]",
	Short: "Creates a new Linux command under a category or sub-category",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		commandName := args[0]
		categoryName := args[1]

		// Read existing YAML
		data, err := ioutil.ReadFile("commands.yaml")
		if err != nil {
			panic(err)
		}

		var config Config
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			panic(err)
		}

		// Find the category path
		categoryPath, err := findCategoryPath(categoryName, config.Categories)
		if err != nil {
			panic(err)
		}

		// Add new command and update YAML
		_, err = addNewCommandToYAML(commandName, categoryName, "linux", &config.Categories)
		if err != nil {
			panic(err)
		}

		// Create the shell script
		err = createShellScript(categoryPath, commandName, "")
		if err != nil {
			panic(err)
		}

		// Update YAML file
		data, err = yaml.Marshal(&config)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile("commands.yaml", data, 0644)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Created new Linux command: %s in category: %s\n", commandName, categoryName)
	},
}

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compiles all .go files",
	Run: func(cmd *cobra.Command, args []string) {
		// Read existing YAML
		data, err := ioutil.ReadFile("commands.yaml")
		if err != nil {
			panic(err)
		}

		var config Config
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			panic(err)
		}

		for _, category := range config.Categories {
			compileAllGoFiles(category.Path)
		}
	},
}

var removeCommandCmd = &cobra.Command{
	Use:   "remove [name] [category]",
	Short: "Removes a command from a category or sub-category",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		commandName := args[0]
		categoryName := args[1]

		// Read existing YAML
		data, err := ioutil.ReadFile("commands.yaml")
		if err != nil {
			panic(err)
		}

		var config Config
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			panic(err)
		}

		// Find the parent category
		parentCategory, err := findParentCategory(categoryName, config.Categories)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}

		// Remove command and update YAML
		err = removeCommandFromYAML(commandName, parentCategory.Name, &config.Categories)
		if err != nil {
			panic(err)
		}

		// Update YAML file
		data, err = yaml.Marshal(&config)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile("commands.yaml", data, 0644)
		if err != nil {
			panic(err)
		}

		// Remove the corresponding .go or .sh file
		filePath := filepath.Join(parentCategory.Path, commandName+".go")
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			err = os.Remove(filePath)
			if err != nil {
				fmt.Printf("Error removing file %s: %s\n", filePath, err.Error())
				return
			}
		} else {
			filePath = filepath.Join(parentCategory.Path, commandName+".sh")
			err = os.Remove(filePath)
			if err != nil {
				fmt.Printf("Error removing file %s: %s\n", filePath, err.Error())
				return
			}
		}

		fmt.Printf("Removed command: %s from category: %s\n", commandName, categoryName)
	},
}

var generateGoCommandCmd = &cobra.Command{
	Use:   "generate-go-command [name] [prompt]",
	Short: "Generates a new Go command based on a GPT-3 prompt",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		commandName := args[0]
		prompt := args[1]

		// Send the prompt to GPT-4 and get the generated code
		generatedCode, err := sendPromptToGPT4(fmt.Sprintf("make command %s %s, please provide the answer as pure code which can be put in a .go file for compilation.", commandName, prompt))
		if err != nil {
			panic(err)
		}

		// Create .go file with the generated code
		filePath := filepath.Join("./generated-commands", commandName+".go")
		err = ioutil.WriteFile(filePath, []byte(generatedCode), 0644)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Generated Go command: %s based on prompt: %s\n", commandName, prompt)
	},
}

var generateLinuxCommandCmd = &cobra.Command{
	Use:   "generate-linux-command [name] [prompt]",
	Short: "Generates a new Linux command based on a GPT-4 prompt",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		commandName := args[0]
		prompt := args[1]

		// Send the prompt to GPT-4 and get the generated code
		generatedCode, err := sendPromptToGPT4(fmt.Sprintf("make Linux command %s %s, please provide the answer as pure code which can be put in a .sh file for execution.", commandName, prompt))
		if err != nil {
			panic(err)
		}

		// Create .sh file with the generated code
		filePath := filepath.Join("./generated-commands", commandName+".sh")
		err = ioutil.WriteFile(filePath, []byte(generatedCode), 0755) // 0755 to make it executable
		if err != nil {
			panic(err)
		}

		fmt.Printf("Generated Linux command: %s based on prompt: %s\n", commandName, prompt)
	},
}

var completionCmd = &cobra.Command{
	Use:                   "completion [bash|zsh|fish|powershell]",
	Short:                 "Generate completion script",
	Long:                  "To load completions",
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			err := cmd.Root().GenBashCompletion(os.Stdout)
			if err != nil {
				fmt.Println("Error creating auto-complete files:", err)
				return
			}
		case "zsh":
			err := cmd.Root().GenZshCompletion(os.Stdout)
			if err != nil {
				fmt.Println("Error creating auto-complete files:", err)
				return
			}
		case "fish":
			err := cmd.Root().GenFishCompletion(os.Stdout, true)
			if err != nil {
				fmt.Println("Error creating auto-complete files:", err)
				return
			}
		case "powershell":
			fmt.Println("Error creating auto-complete files:", "asd")
			err := cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			fmt.Println("Error creating auto-complete files:", err)
			if err != nil {
				fmt.Println("Error creating auto-complete files:", err)
				return
			}
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list [category] [subCategory1] [subCategory2] ...",
	Short: "Lists built-in commands, categories, and commands for a given category and subcategories",
	Run: func(cmd *cobra.Command, args []string) {
		// Read existing YAML
		data, err := ioutil.ReadFile("commands.yaml")
		if err != nil {
			panic(err)
		}

		var config Config
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			panic(err)
		}

		listCommandsAndCategories(args, config.Categories)
	},
}

// addNewGoCommandToCategory adds a new Go command to a given category,
// updates the commands.yaml file, and creates a new .go file for the command.
func addNewGoCommandToCategory(commandName string, parentCategory *Category, config *Config) error {
	// Add the new command to the parent category
	newCommand := Command{
		Name:      commandName,
		Extension: ".exe",
	}
	parentCategory.Commands = append(parentCategory.Commands, newCommand)

	// Update the commands.yaml file
	updatedData, err := yaml.Marshal(&config) // Assume 'config' contains the updated categories
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("commands.yaml", updatedData, 0644)
	if err != nil {
		return err
	}

	// Create a new .go file for the command
	goFilePath := filepath.Join(parentCategory.Path, fmt.Sprintf("%s.go", commandName))

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(goFilePath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		err = os.MkdirAll(parentDir, 0755)
		if err != nil {
			return err
		}
	}

	goFileContent := `package main

import "fmt"

func main() {
    fmt.Println("Hello, this is ` + commandName + `!")
}
`
	err = ioutil.WriteFile(goFilePath, []byte(goFileContent), 0644)
	if err != nil {
		return err
	}

	return nil
}

func addToPath() error {
	// Get the path of the currently running executable
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exePath)

	switch runtime.GOOS {
	case "windows":
		println("halp")
		// Windows-specific code to add to PATH using batch commands
		cmd := exec.Command("setx", "PATH", "%PATH%;"+exeDir)
		err := cmd.Run()
		if err != nil {
			return err
		}
	case "linux", "darwin":
		// Unix-specific code to add to PATH using shell commands
		// This will add the path to the current session. To make it permanent, you'd have to add it to .bashrc, .zshrc, etc.
		shellFiles := []string{"~/.bashrc", "~/.zshrc"}
		for _, shellFile := range shellFiles {
			cmd := exec.Command("bash", "-c", `echo 'export PATH=$PATH:`+exeDir+`' >> `+shellFile)
			err := cmd.Run()
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return nil
}

// findParentCategory searches for a category or subcategory by name
// using a breadth-first search approach.
func findParentCategory(name string, categories []Category) (*Category, error) {
	// Initialize a queue with the root categories
	queue := make([]*Category, len(categories))
	for i := range categories {
		queue[i] = &categories[i]
	}

	// Perform BFS
	for len(queue) > 0 {
		var nextQueue []*Category
		for _, category := range queue {
			if category.Name == name {
				return category, nil
			}
			for i := range category.Subcategories {
				nextQueue = append(nextQueue, &category.Subcategories[i])
			}
		}
		queue = nextQueue
	}

	return nil, errors.New("category not found")
}

func createCategoryFolder(categoryName string, parentCategoryPath string) error {
	newFolderPath := filepath.Join(parentCategoryPath, categoryName)
	err := os.MkdirAll(newFolderPath, 0755)
	if err != nil {
		return err
	}
	return nil
}

func updateYAMLWithNewCategory(categoryName string, parentCategoryName string) error {
	// Read existing YAML
	data, err := ioutil.ReadFile("commands.yaml")
	if err != nil {
		return err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	if parentCategoryName == "" {
		// Check if category already exists at root level
		for _, category := range config.Categories {
			if category.Name == categoryName {
				return fmt.Errorf("category %s already exists", categoryName)
			}
		}
		newCategory := Category{
			Name: categoryName,
			Path: filepath.Join(".", categoryName),
		}
		config.Categories = append(config.Categories, newCategory)
	} else {
		found := false
		var queue []*Category
		for i := range config.Categories {
			queue = append(queue, &config.Categories[i])
		}

		for len(queue) > 0 {
			var nextQueue []*Category
			for _, category := range queue {
				if category.Name == parentCategoryName {
					// Check if subcategory already exists
					for _, subCategory := range category.Subcategories {
						if subCategory.Name == categoryName {
							return fmt.Errorf("subcategory %s already exists under %s", categoryName, parentCategoryName)
						}
					}
					newCategory := Category{
						Name: categoryName,
						Path: filepath.Join(category.Path, categoryName),
					}
					category.Subcategories = append(category.Subcategories, newCategory)
					found = true
					break
				}
				for i := range category.Subcategories {
					nextQueue = append(nextQueue, &category.Subcategories[i])
				}
			}
			if found {
				break
			}
			queue = nextQueue
		}

		if !found {
			return fmt.Errorf("Parent category not found")
		}
	}

	// Write updated YAML back to file
	updatedData, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("commands.yaml", updatedData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func findCategoryPath(categoryName string, categories []Category) (string, error) {
	for _, category := range categories {
		if category.Name == categoryName {
			return category.Path, nil
		}
		if len(category.Subcategories) > 0 {
			path, err := findCategoryPath(categoryName, category.Subcategories)
			if err == nil {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("category not found")
}

func sendPromptToGPT4(prompt string) (string, error) {
	apiURL := "https://api.openai.com/v1/engines/davinci-codex/completions" // Replace with actual GPT-4 API URL
	apiKey := "your-api-key-here"

	reqBody := GPT4Request{
		Prompt:    prompt,
		MaxTokens: 100,
	}
	jsonReq, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonReq))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Failed to close Body: %s\n", err.Error())
		}
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var gpt4Resp GPT4Response
	err = json.Unmarshal(body, &gpt4Resp)
	if err != nil {
		return "", err
	}

	return gpt4Resp.Choices[0].Text, nil
}

func initializeAutoComplete() {
	newLinuxCommandCmd.Flags().StringVarP(&scriptContent, "content", "c", "", "Optional content for the .sh file")

	f := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Read existing YAML
		data, err := ioutil.ReadFile("commands.yaml")
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var config Config
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		// Generate a list of existing category names for auto-completion

		println(len(args))

		if len(args) == 1 {
			return getAllCategoryNames(config), cobra.ShellCompDirectiveDefault
		}

		return append(make([]string, 1), ""), cobra.ShellCompDirectiveDefault
	}

	newCategoryCmd.ValidArgsFunction = f
	generateGoCommandCmd.ValidArgsFunction = f
	generateLinuxCommandCmd.ValidArgsFunction = f
}

func executeProgram(program string, args []string) {
	cmd := exec.Command(program, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Failed to execute program %s: %s\n", program, err.Error())
		return
	}
}

func addCommandsToCategory(catCmd *cobra.Command, category Category) {
	for _, command := range category.Commands {
		// Construct the full executable path using the extension
		executablePath := filepath.Join(category.Path, command.Name+command.Extension)

		cmd := &cobra.Command{
			Use:   command.Name,
			Short: "Runs the " + command.Name + " executable",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Printf("%s: %s\n", command.Name, executablePath)
				// Using filepath.Join to ensure the path is correctly formed
				fullPath := filepath.Join(".", executablePath)
				executeProgram(fullPath, args)
			},
		}
		catCmd.AddCommand(cmd)
	}

	for _, subcategory := range category.Subcategories {
		subCatCmd := &cobra.Command{
			Use:   subcategory.Name,
			Short: "Commands under " + subcategory.Name,
		}
		addCommandsToCategory(subCatCmd, subcategory)
		catCmd.AddCommand(subCatCmd)
	}
}

func createGoFile(path string, commandName string) error {
	content := []byte(fmt.Sprintf(`package main

import "fmt"

func main() {
	fmt.Println("Running %s program")
}
`, commandName))

	filePath := filepath.Join(path, commandName+".go")
	return ioutil.WriteFile(filePath, content, 0644)
}

func addNewCommandToYAML(commandName string, categoryName string, commandType string, categories *[]Category) (string, error) {
	var newCommand Command
	if commandType == "go" {
		newCommand = Command{
			Name:      commandName,
			Extension: ".exe",
		}
	} else if commandType == "linux" {
		newCommand = Command{
			Name:      commandName,
			Extension: ".sh",
		}
	} else {
		print(fmt.Errorf("empty command type for command %s\n", commandName))
	}

	for i, category := range *categories {
		if category.Name == categoryName {
			(*categories)[i].Commands = append((*categories)[i].Commands, newCommand)
			if commandType == "go" {
				return category.Path, createGoFile(category.Path, commandName)
			} else {
				return category.Path, createShellScript(category.Path, commandName, scriptContent)
			}
		}

		if len(category.Subcategories) > 0 {
			path, err := addNewCommandToYAML(commandName, categoryName, commandType, &(*categories)[i].Subcategories)
			if err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("category not found")
}

func createShellScript(path string, commandName string, content string) error {
	if content == "" {
		content = fmt.Sprintf(`#!/bin/bash

echo "Running %s Linux command"
`, commandName)
	}

	filePath := filepath.Join(path, commandName+".sh")
	return ioutil.WriteFile(filePath, []byte(content), 0755)
}

func compileAllGoFiles(path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println("Failed to read directory:", err)
		return
	}

	done := make(chan bool)

	for _, file := range files {
		filePath := filepath.Join(path, file.Name())
		if file.IsDir() {
			compileAllGoFiles(filePath)
		} else if strings.HasSuffix(file.Name(), ".go") {
			go compileGoFile(filePath, done)
			<-done
		}
	}
}

func compileGoFile(filePath string, done chan bool) {
	cmd := exec.Command("go", "build", "-o", filePath[:len(filePath)-3]+".exe", filePath)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Failed to compile %s: %s\n", filePath, err)
	} else {
		fmt.Printf("Successfully compiled %s\n", filePath)
	}
	if done != nil {
		done <- true
	}
}

func removeCommandFromYAML(commandName string, categoryName string, categories *[]Category) error {
	for i, category := range *categories {
		if category.Name == categoryName {
			for j, cmd := range category.Commands {
				if cmd.Name == commandName {
					// Remove the command from the list
					(*categories)[i].Commands = append((*categories)[i].Commands[:j], (*categories)[i].Commands[j+1:]...)
					return nil
				}
			}
		}

		if len(category.Subcategories) > 0 {
			err := removeCommandFromYAML(commandName, categoryName, &(*categories)[i].Subcategories)
			if err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("command or category not found")
}

func getAllCategoryNames(config Config) []string {
	var categories []string
	for _, category := range config.Categories {
		categories = append(categories, collectCategories(category)...)
	}
	return categories
}

func collectCategories(category Category) []string {
	var result []string
	result = append(result, category.Name)
	for _, subcategory := range category.Subcategories {
		result = append(result, collectCategories(subcategory)...)
	}
	return result
}

func listCommandsAndCategories(categoryNames []string, categories []Category) {
	if len(categoryNames) == 0 {
		fmt.Println("Built-in Commands:")
		// List built-in commands here (replace with your actual list)
		fmt.Println(
			"new-category\nnew-go-command\n" +
				"new-linux-command\ncompile\n" +
				"remove\ngenerate-go-command\n" +
				"generate-linux-command\nlist")

		fmt.Println("\nCategories:")
		for _, category := range categories {
			fmt.Println("  " + category.Name)
		}
		return
	}

	currentCategory := categoryNames[0]
	for _, category := range categories {
		if category.Name == currentCategory {
			if len(categoryNames) == 1 {
				fmt.Println("\nCommands in " + currentCategory + ":")
				for _, cmd := range category.Commands {
					fmt.Println("  " + cmd.Name)
				}
				if len(category.Subcategories) > 0 {
					fmt.Println("\nSubcategories in " + currentCategory + ":")
					for _, subcat := range category.Subcategories {
						fmt.Println("  " + subcat.Name)
					}
				}
			} else {
				listCommandsAndCategories(categoryNames[1:], category.Subcategories)
			}
			return
		}
		if len(category.Subcategories) > 0 {
			listCommandsAndCategories(categoryNames, category.Subcategories)
		}
	}
}

func detectShell() string {
	shellEnv := os.Getenv("SHELL")

	// Try running a PowerShell command to check if PSVersionTable exists
	cmd := exec.Command("powershell", "-Command", "if ($null -ne $PSVersionTable) { Write-Host 'PowerShell' } else { Write-Host 'Unknown' }")
	output, err := cmd.CombinedOutput()
	if err == nil && strings.TrimSpace(string(output)) == "PowerShell" {
		return "powershell"
	}

	if strings.Contains(shellEnv, "bash") {
		return "bash"
	} else if strings.Contains(shellEnv, "zsh") {
		return "zsh"
	} else if strings.Contains(shellEnv, "fish") {
		return "fish"
	}

	// For Windows Command Prompt
	shellCmd := os.Getenv("COMSPEC")
	if strings.Contains(strings.ToLower(shellCmd), "cmd.exe") {
		return "cmd"
	}

	return "unknown"
}

func Initialize() {
	// Initialize commands.yaml with a hello-world command
	config := Config{}

	// Write to commands.yaml
	data, err := yaml.Marshal(&config)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("commands.yaml", data, 0644)
	if err != nil {
		panic(err)
	}

	err = addToPath()
	if err != nil {
		panic(err)
	}

	fmt.Println("Initialized")
}

func main() {
	var rootCmd = &cobra.Command{Use: "asd"}

	// Check if commands.yaml exists
	if _, err := os.Stat("commands.yaml"); os.IsNotExist(err) {
		Initialize()
		return
	}

	// Read YAML file
	data, err := ioutil.ReadFile("commands.yaml")
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	for _, category := range config.Categories {
		catCmd := &cobra.Command{
			Use:   category.Name,
			Short: "Commands under " + category.Name,
		}
		addCommandsToCategory(catCmd, category)
		rootCmd.AddCommand(catCmd)
	}

	initializeAutoComplete()

	rootCmd.AddCommand(
		listCmd,
		newCategoryCmd,
		newGoCommandCmd,
		newLinuxCommandCmd,
		compileCmd,
		removeCommandCmd,
		generateGoCommandCmd,
		generateLinuxCommandCmd,
		completionCmd,
	)

	// Add shell completion
	shell := detectShell()
	switch shell {
	case "powershell":
		println("1")
		cmd := exec.Command("powershell", "-Command", "asd completion powershell | Out-String | Invoke-Expression")
		err := cmd.Run()
		if err != nil {
			fmt.Printf("cmd.Run() failed with %s\n", err)
		}
		break
	case "bash":
		println("2")
		runCommand("source <(asd completion bash)")
		break
	case "zsh":
		println("3")
		runCommand("source <(asd completion zsh)")
		break
	case "fish":
		println("4")
		runCommand("asd completion fish | source")
		break
	default:
		println("5")
		fmt.Printf("The shell '%s' can not load auto-completion, continuing without auto-completion.", shell)
		break
	}

	err = rootCmd.Execute()
	if err != nil {
		fmt.Printf("Failed to execute root command: %s\n", err.Error())
		return
	}
}

func runCommand(command string) {
	cmd := exec.Command(command)

	println(command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}

	fmt.Printf(" ::: Auto-complete status :::\n%s\n", output)
}
