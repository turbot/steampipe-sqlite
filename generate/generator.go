//go:build tool

package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const templateExt = ".tmpl"

func RenderDir(templatePath, root, pluginAlias, pluginGithubUrl string) {
	var targetFilePath string
	err := filepath.Walk(templatePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %s: %v\n", filePath, err)
			return nil
		}

		fmt.Println("filePath:", filePath)
		if info.IsDir() {
			// fmt.Println("not a file, continuing...\n")
			return nil
		}

		relativeFilePath := strings.TrimPrefix(filePath, root)
		// fmt.Println("relative path:", relativeFilePath)
		ext := filepath.Ext(filePath)
		// fmt.Println("extension:", ext)

		if ext != templateExt {
			// fmt.Println("not tmpl, continuing...\n")
			return nil
		}

		templateFileName := strings.TrimPrefix(relativeFilePath, "/templates/")
		// fmt.Println("template fileName:", templateFileName)
		fileName := strings.TrimSuffix(templateFileName, ext)
		// fmt.Println("actual fileName:", fileName)

		targetFilePath = path.Join(root, fileName)
		// fmt.Println("targetFilePath:", targetFilePath)

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
		data := struct {
			PluginAlias     string
			PluginGithubUrl string
		}{
			pluginAlias,
			pluginGithubUrl,
		}

		// execute the template with the data
		if err := tmpl.Execute(&renderedContent, data); err != nil {
			fmt.Printf("Error rendering template: %v\n", err)
			return err
		}

		if err := os.MkdirAll(filepath.Dir(targetFilePath), 0755); err != nil {
			fmt.Printf("Error creating directory: %v\n", err)
			return err
		}

		// write the rendered content to the target file
		if err := os.WriteFile(targetFilePath, []byte(renderedContent.String()), 0644); err != nil {
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
	if len(os.Args) != 5 {
		fmt.Println("Usage: go run generator.go <templatePath> <root> <pluginAlias> <pluginGithubUrl>")
		return
	}

	templatePath := os.Args[1]
	root := os.Args[2]
	pluginAlias := os.Args[3]
	pluginGithubUrl := os.Args[4]

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

	RenderDir(absTemplatePath, absRoot, pluginAlias, pluginGithubUrl)
}
