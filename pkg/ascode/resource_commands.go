package ascode

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kataras/golog"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// ResourceImporter defines the interface for importing resources
type ResourceImporter interface {
	ImportResource(yaml string) error
}

// ApplyResourceCmd creates a Cobra command for applying a resource from a file
func ApplyResourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "apply",
		Short:            "Imports a resource from a file or a folder",
		Example:          `apply resource.yaml`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("expected exactly one argument, the file or folder to import")
			}

			fileOrFolderPath := args[0]
			if _, err := os.Stat(fileOrFolderPath); os.IsNotExist(err) {
				return fmt.Errorf("file or folder %s does not exist", fileOrFolderPath)
			}

			importer := config.Client

			if err := applyResource(importer, fileOrFolderPath); err != nil {
				return fmt.Errorf("failed to apply resource: %s", err)
			}

			return nil
		},
	}

	return cmd
}

func applyResource(importer ResourceImporter, path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return applyResourcesFromFolder(importer, path)
	}

	return applyResourceFile(importer, path)
}

func applyResourceFile(importer ResourceImporter, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %s", err)
	}

	fileContent := string(content)
	var resource Resource
	if err := yaml.Unmarshal(content, &resource); err != nil {
		return fmt.Errorf("invalid YAML file content: %s", err)
	}

	apiVersion := resource.APIVersion
	kind := resource.Kind

	if apiVersion != "lenses.io/v0beta" {
		return fmt.Errorf("unsupported API version: %s. Only supporting 'lenses.io/v0beta'", apiVersion)
	}

	switch kind {
	case "KafkaConnector":
		if err := importer.ImportResource(fileContent); err != nil {
			return fmt.Errorf("failed to import KafkaConnector resource: %s", err)
		}
	default:
		return fmt.Errorf("unsupported kind: %s. Supported kind: KafkaConnector", kind)
	}

	return nil
}

func applyResourcesFromFolder(importer ResourceImporter, folderPath string) error {
	err := filepath.Walk(folderPath, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if file == folderPath {
			return nil
		}
		if !info.IsDir() {
			golog.Infof("Processing file:%s", file)

			if err := applyResourceFile(importer, file); err != nil {
				return fmt.Errorf("failed to apply resource from file %s: %s", file, err)
			}
		} else {
			golog.Infof("Processing folder: %", file)
			err = applyResourcesFromFolder(importer, file)
			if err != nil {
				return fmt.Errorf("failed to apply resource from folder %s: %s", file, err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking through folder %s: %s", folderPath, err)
	}

	return nil
}

// Resource struct definition
type Resource struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	// Add other fields as needed
}
