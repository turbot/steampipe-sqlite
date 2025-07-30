//go:build tool

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

const templateExt = ".tmpl"

type RenderData struct {
	Plugin          string
	PluginGithubUrl string
	PluginVersion   string
	GoVersion       string
	GoToolchain     string
}

type ModuleInfo struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Dir     string `json:"Dir"`
}

type GoVersionInfo struct {
	Version   string // Go version (e.g., "1.23.1")
	Toolchain string // Go toolchain (e.g., "go1.24.1")
}

// extractGoVersionInfo extracts Go version and toolchain information from go.mod content
func extractGoVersionInfo(content string) GoVersionInfo {
	var info GoVersionInfo
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			info.Version = strings.TrimSpace(strings.TrimPrefix(line, "go"))
		} else if strings.HasPrefix(line, "toolchain ") {
			info.Toolchain = strings.TrimSpace(strings.TrimPrefix(line, "toolchain"))
		}
	}
	return info
}

// validateGoVersionInfo validates the Go version format
func validateGoVersionInfo(info GoVersionInfo) error {
	if info.Version != "" {
		versionRegex := regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)
		if !versionRegex.MatchString(info.Version) {
			return fmt.Errorf("invalid Go version format: %s", info.Version)
		}
	}
	if info.Toolchain != "" {
		toolchainRegex := regexp.MustCompile(`^go\d+\.\d+\.\d+$`)
		if !toolchainRegex.MatchString(info.Toolchain) {
			return fmt.Errorf("invalid Go toolchain format: %s", info.Toolchain)
		}
	}
	return nil
}

// validateGithubUrl validates that the URL is a valid GitHub repository URL
func validateGithubUrl(urlStr string) error {
	if !strings.HasPrefix(urlStr, "github.com/") {
		return fmt.Errorf("URL must be a GitHub repository URL (github.com/...)")
	}
	parts := strings.Split(urlStr, "/")
	if len(parts) != 3 {
		return fmt.Errorf("URL must be in format github.com/owner/repo")
	}
	return nil
}

// validateModulePath validates a Go module path format
func validateModulePath(path string) error {
	// Module path should be alphanumeric with possible dots, dashes, and slashes
	re := regexp.MustCompile(`^[\w\-\.]+(/[\w\-\.]+)*$`)
	if !re.MatchString(path) {
		return fmt.Errorf("invalid module path format: %s", path)
	}
	return nil
}

// validateVersion validates a Go module version format
func validateVersion(version string) error {
	if version == "" {
		return nil // Version is optional
	}
	// Version should be in format v1.2.3 or v1.2.3-beta.1
	re := regexp.MustCompile(`^v\d+\.\d+\.\d+(-[\w\-\.]+)*$`)
	if !re.MatchString(version) {
		return fmt.Errorf("invalid version format: %s", version)
	}
	return nil
}

// validateReplaceDirective validates a replace directive format
func validateReplaceDirective(directive string) error {
	// Split into original module and replacement
	parts := strings.Split(strings.TrimSpace(directive), "=>")
	if len(parts) != 2 {
		return fmt.Errorf("invalid replace directive format (should be 'module => replacement [version]'): %s", directive)
	}

	// Validate original module path
	origModule := strings.TrimSpace(parts[0])
	if err := validateModulePath(origModule); err != nil {
		return fmt.Errorf("invalid original module: %v", err)
	}

	// Split replacement into module and version
	replacement := strings.TrimSpace(parts[1])
	repParts := strings.Fields(replacement)
	if len(repParts) == 0 || len(repParts) > 2 {
		return fmt.Errorf("invalid replacement format (should be 'module [version]'): %s", replacement)
	}

	// Validate replacement module path
	if err := validateModulePath(repParts[0]); err != nil {
		return fmt.Errorf("invalid replacement module: %v", err)
	}

	// Validate version if present
	if len(repParts) == 2 {
		if err := validateVersion(repParts[1]); err != nil {
			return fmt.Errorf("invalid replacement version: %v", err)
		}
	}

	return nil
}

func getPluginReplaceDirectives(pluginGithubUrl string, targetDir string) (string, GoVersionInfo, error) {
	// Validate GitHub URL
	if err := validateGithubUrl(pluginGithubUrl); err != nil {
		return "", GoVersionInfo{}, fmt.Errorf("invalid GitHub URL: %v", err)
	}

	fmt.Printf("Getting replace directives for plugin: %s\n", pluginGithubUrl)

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "plugin-mod")
	if err != nil {
		return "", GoVersionInfo{}, fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("Created temp dir: %s\n", tmpDir)

	// Initialize a go module
	cmd := exec.Command("go", "mod", "init", "temp")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		return "", GoVersionInfo{}, fmt.Errorf("failed to init module: %v", err)
	}

	fmt.Printf("Initialized temp module\n")

	// Extract owner and repo from URL for safer command execution
	urlParts := strings.Split(pluginGithubUrl, "/")
	if len(urlParts) != 3 {
		return "", GoVersionInfo{}, fmt.Errorf("invalid GitHub URL format")
	}
	safeUrl := fmt.Sprintf("github.com/%s/%s", urlParts[1], urlParts[2])

	// Get the latest version of the plugin
	cmd = exec.Command("go", "get")
	cmd.Dir = tmpDir
	cmd.Args = append(cmd.Args, "--", safeUrl) // Ensure URL is treated as literal
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", GoVersionInfo{}, fmt.Errorf("failed to get plugin: %v\nOutput: %s", err, output)
	}

	fmt.Printf("Got plugin\n")

	// Get the module info
	cmd = exec.Command("go", "list", "-m", "-json")
	cmd.Dir = tmpDir
	cmd.Args = append(cmd.Args, "--", safeUrl) // Ensure URL is treated as literal
	output, err := cmd.Output()
	if err != nil {
		return "", GoVersionInfo{}, fmt.Errorf("failed to get module info: %v", err)
	}

	fmt.Printf("Got module info: %s\n", output)

	var info ModuleInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return "", GoVersionInfo{}, fmt.Errorf("failed to parse module info: %v", err)
	}

	fmt.Printf("Module dir: %s\n", info.Dir)

	// Read the go.mod file
	goModPath := filepath.Join(info.Dir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", GoVersionInfo{}, fmt.Errorf("failed to read go.mod: %v", err)
	}

	fmt.Printf("Read go.mod: %s\n", content)

	// Extract Go version info
	goInfo := extractGoVersionInfo(string(content))
	if err := validateGoVersionInfo(goInfo); err != nil {
		return "", GoVersionInfo{}, fmt.Errorf("invalid Go version info in plugin go.mod: %v", err)
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
				// Validate replace directive
				if err := validateReplaceDirective(line); err != nil {
					return "", GoVersionInfo{}, fmt.Errorf("invalid replace directive in plugin go.mod: %v", err)
				}
				replaces = append(replaces, line)
			}
		} else if strings.HasPrefix(line, "replace ") {
			directive := strings.TrimPrefix(line, "replace ")
			// Validate replace directive
			if err := validateReplaceDirective(directive); err != nil {
				return "", GoVersionInfo{}, fmt.Errorf("invalid replace directive in plugin go.mod: %v", err)
			}
			replaces = append(replaces, directive)
		}
	}

	fmt.Printf("Found replace directives: %v\n", replaces)

	var replaceContent string
	if len(replaces) > 0 {
		// Create replace directives content
		var content strings.Builder
		content.WriteString("\n// Replace directives from plugin\nreplace (\n")
		for _, r := range replaces {
			content.WriteString("\t")
			content.WriteString(r)
			content.WriteString("\n")
		}
		content.WriteString(")\n")
		replaceContent = content.String()

		// Write to replace.mod file in the target directory
		replaceModPath := filepath.Join(targetDir, "replace.mod")
		if err := os.WriteFile(replaceModPath, []byte(replaceContent), 0644); err != nil {
			return "", GoVersionInfo{}, fmt.Errorf("failed to write replace directives: %v", err)
		}
	}

	return replaceContent, goInfo, nil
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

		// If this is go.mod, get plugin info first
		var goInfo GoVersionInfo
		var replaces string
		if strings.HasSuffix(targetFilePath, "go.mod") {
			var err error
			replaces, goInfo, err = getPluginReplaceDirectives(pluginGithubUrl, root)
			if err != nil {
				fmt.Printf("Error getting plugin replace directives: %v\n", err)
			}
		}

		// define the data to be used in the template
		data := RenderData{
			Plugin:          pluginAlias,
			PluginGithubUrl: pluginGithubUrl,
			GoVersion:       goInfo.Version,
			GoToolchain:     goInfo.Toolchain,
		}

		// execute the template with the data
		if err := tmpl.Execute(&renderedContent, data); err != nil {
			fmt.Printf("Error rendering template: %v\n", err)
			return err
		}

		content := renderedContent.String()

		// If this is go.mod, append any replace directives
		if strings.HasSuffix(targetFilePath, "go.mod") && replaces != "" {
			content += replaces
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
