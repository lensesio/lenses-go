package export

import (
	"io/fs"
	"os"
)

type OSFileWriter struct{}

// MkdirAll creates the landscape dir with the specified mode
func (w OSFileWriter) MkdirAll(landscape string, mode fs.FileMode) error {
	return os.MkdirAll(landscapeDir, mode)
}

// WriteFile writes the content to a file with the specified filename and mode.
func (w OSFileWriter) WriteFile(filePath string, content string, mode fs.FileMode) error {
	return os.WriteFile(filePath, []byte(content), mode)
}
