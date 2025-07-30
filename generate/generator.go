//go:build tool

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const templateExt = ".tmpl"

type RenderData struct {
	Plugin          string
	PluginGithubUrl string
	PluginVersion   string
}

type ModuleInfo struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Dir     string `json:"Dir"`
}

func getPluginReplaceDirectives(pluginGithubUrl string) (string, error) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "plugin-mod")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a go module
	cmd := exec.Command("go", "mod", "init", "temp")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to init module: %v", err)
	}

	// Get the latest version of the plugin
	cmd = exec.Command("go", "get", pluginGithubUrl)
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get plugin: %v", err)
	}

	// Get the module info
	cmd = exec.Command("go", "list", "-m", "-json", pluginGithubUrl)
	cmd.Dir = tmpDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get module info: %v", err)
	}

	var info ModuleInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return "", fmt.Errorf("failed to parse module info: %v", err)
	}

	// Read the go.mod file
	goModPath := filepath.Join(info.Dir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %v", err)
	}

	// Extract replace directives
	var replaces []string
	lines := strings.Split(string(content), "\n")
	inReplace := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "replace (") {
			inReplace = true
			continue
		}
		if inReplace {
			if line == ")" {
				inReplace = false
				continue
			}
			if line != "" {
				replaces = append(replaces, line)
			}
		} else if strings.HasPrefix(line, "replace ") {
			replaces = append(replaces, strings.TrimPrefix(line, "replace "))
		}
	}

	if len(replaces) > 0 {
		return "\n// Replace directives from plugin\nreplace (\n\t" + strings.Join(replaces, "\n\t") + "\n)\n", nil
	}
	return "", nil
}

func RenderDir(templatePath, root, pluginAlias, pluginGithubUrl string) {
	var targetFilePath string
	err := filepath.Walk(templatePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %s: %v\n", filePath, err)
			return nil
		}

		fmt.Println("filePath:", filePath)
		if info.IsDir() {
			return nil
		}

		relativeFilePath := strings.TrimPrefix(filePath, root)
		ext := filepath.Ext(filePath)

		if ext != templateExt {
			return nil
		}

		templateFileName := strings.TrimPrefix(relativeFilePath, "/templates/")
		fileName := strings.TrimSuffix(templateFileName, ext)
		targetFilePath = path.Join(root, fileName)

		// read template file
		templateContent, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading template file: %v\n", err)
			return err
		}

		// create a new template and parse the content
		tmpl := template.Must(template.New(targetFilePath).Parse(string(templateContent)))

		// create a buffer to render the template
		var renderedContent strings.Builder

		// define the data to be used in the template
		data := RenderData{
			Plugin:          pluginAlias,
			PluginGithubUrl: pluginGithubUrl,
		}

		// execute the template with the data
		if err := tmpl.Execute(&renderedContent, data); err != nil {
			fmt.Printf("Error rendering template: %v\n", err)
			return err
		}

		content := renderedContent.String()

		// If this is go.mod, append any replace directives from the plugin
		if strings.HasSuffix(targetFilePath, "go.mod") {
			if replaces, err := getPluginReplaceDirectives(pluginGithubUrl); err == nil && replaces != "" {
				content += replaces
			}
		}

		if err := os.MkdirAll(filepath.Dir(targetFilePath), 0755); err != nil {
			fmt.Printf("Error creating directory: %v\n", err)
			return err
		}

		// write the rendered content to the target file
		if err := os.WriteFile(targetFilePath, []byte(content), 0644); err != nil {
			fmt.Printf("Error writing to target file: %v\n", err)
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Println(err)
		return
	}
}

func main() {
	// Check if the correct number of command-line arguments are provided
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run generator.go <templatePath> <root> <plugin> [pluginGithubUrl]")
		return
	}

	templatePath := os.Args[1]
	root := os.Args[2]
	plugin := os.Args[3]
	var pluginGithubUrl string

	// Check if PluginGithubUrl is provided as a command-line argument
	if len(os.Args) >= 5 {
		pluginGithubUrl = os.Args[4]
	} else {
		// If PluginGithubUrl is not provided, generate it based on PluginAlias
		pluginGithubUrl = "github.com/turbot/steampipe-plugin-" + plugin
	}

	// Convert relative paths to absolute paths
	absTemplatePath, err := filepath.Abs(templatePath)
	if err != nil {
		fmt.Printf("Error converting templatePath to absolute path: %v\n", err)
		return
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Printf("Error converting root to absolute path: %v\n", err)
		return
	}

	RenderDir(absTemplatePath, absRoot, plugin, pluginGithubUrl)
}
