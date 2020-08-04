package utils

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/lenses-go/pkg/api"
	"gopkg.in/yaml.v2"
)

//CreateDirectory creates a directory with full permissions
func CreateDirectory(directoryPath string) error {
	return os.MkdirAll(directoryPath, 0777)
}

//DecryptAES decrypting AES
func decryptAES(key, h []byte) ([]byte, error) {
	iv := h[:aes.BlockSize]
	h = h[aes.BlockSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	str := cipher.NewCFBDecrypter(block, iv)
	str.XORKeyStream(h, h)

	return h, nil
}

//DecryptString descryptin encrypted string with keybase
func DecryptString(encryptedRaw string, keyBase string) (plainTextString string, err error) {
	encrypted, err := base64.URLEncoding.DecodeString(encryptedRaw)
	if err != nil {
		return "", err
	}

	if len(encrypted) < aes.BlockSize {
		return "", fmt.Errorf("short cipher, min len: 16")
	}

	decrypted, err := decryptAES(ToHash(keyBase), encrypted)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

//EncryptAES encrypts data with provided key
func EncryptAES(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	out := make([]byte, aes.BlockSize+len(data))
	iv := out[:aes.BlockSize]
	encrypted := out[aes.BlockSize:]

	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(encrypted, data)

	return out, nil
}

//EncryptString encrypts plain string with the provided keybase (AES)
func EncryptString(plain string, keyBase string) (string, error) {
	key := ToHash(keyBase)
	encrypted, err := EncryptAES(key, []byte(plain))
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(encrypted), nil
}

//Fetch data from a file with a provided prefix
func Fetch(fromFile, prefix string) ([]string, error) {
	var vars []string
	if fromFile != "" {
		golog.Infof("Loading variables from file [%s] with prefix [%s]", fromFile, prefix)
		lines, err := ReadLines(fromFile)

		if err != nil {
			return vars, err
		}

		for _, l := range lines {
			if strings.HasPrefix(l, prefix) {
				vars = append(vars, strings.Replace(strings.Replace(l, prefix, "", 1), "_", ".", -1))
			}
		}
	} else {
		golog.Infof("Looking for environment variables with prefix [%s]", prefix)
		vars = GetEnvVars(prefix)
	}

	if len(vars) == 0 {
		golog.Warnf("No environment variables prefixed with [%s] found or loaded from file [%s]", prefix, fromFile)
	}
	return vars, nil
}

//GetEnvVars returns the environments variables
func GetEnvVars(prefix string) []string {
	var vars []string

	for _, v := range os.Environ() {
		if strings.HasPrefix(v, prefix) {
			golog.Infof("Found environment var [%s]", v)
			split := strings.SplitN(v, "=", 2)

			if len(split) == 2 {
				name := strings.ToLower(strings.Replace(strings.Replace(split[0], prefix, "", 1), "_", ".", -1))
				vars = append(vars, fmt.Sprintf("%s=%s", name, split[1]))
			}
		}
	}

	return vars
}

//FindFiles fidn the files in provided directory
func FindFiles(dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		golog.Fatal(err)
	}
	return files
}

//PrintLogLines prints lines as logs
func PrintLogLines(logs []api.LogLine) error {
	golog.SetTimeFormat("")

	for _, logLine := range logs {
		logLine.Message, _ = url.QueryUnescape(logLine.Message) // for LSQL lines.
		line := logLine.Time + " " + logLine.Message
		RichLog(logLine.Level, line)
	}

	return nil
}

//PrettyPrint prints json with pretty identation
func PrettyPrint(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	return out.Bytes(), err
}

// ReadLines reads a file and returns the string contents
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

//RichLog based on level logs properly
func RichLog(level string, log string) {
	switch strings.ToLower(level) {
	case "info":
		golog.Infof(log)
	case "warn":
		golog.Warnf(log)
	case "error":
		golog.Errorf(log)
	default:
		// app.Print(log)
	}
}

//StringInSlice check if a string is in slice
func StringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

//ToHash hashes with SHA256 the provided string
func ToHash(plain string) []byte {
	h := sha256.Sum256([]byte(plain))
	return h[:]
}

//ToYaml transforms interface data to Yaml
func ToYaml(o interface{}) ([]byte, error) {
	y, err := yaml.Marshal(o)
	return y, err
}

//WalkPropertyValueFromArgs walks the proerty values from arguments
func WalkPropertyValueFromArgs(args []string, actionFunc func(property, value string) error) error {
	if len(args) < 2 {
		return fmt.Errorf("at least two arguments are required, the first is the property name and the second is the actual property's value")
	}

	for i, n := 0, len(args); i < n; i++ {
		property := args[i]
		i++
		if i >= n {
			break
		}
		value := args[i]

		if err := actionFunc(property, value); err != nil {
			return err
		}
	}

	return nil
}

//WriteByteFile writes to a file from byte data
func WriteByteFile(fileName string, data []byte) error {

	os.MkdirAll(filepath.Dir(fileName), os.ModePerm)

	file, err := os.OpenFile(
		fileName,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)

	if err != nil {
		golog.Fatal(err)
		return err
	}
	defer file.Close()

	_, writeErr := file.Write(data)

	if writeErr != nil {
		golog.Fatal(writeErr)
		return writeErr
	}

	return nil
}

//WriteStringFile writes to a file from string data
func WriteStringFile(fileName string, data []string) error {

	os.MkdirAll(filepath.Dir(fileName), os.ModePerm)

	file, err := os.OpenFile(
		fileName,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)

	if err != nil {
		golog.Fatal(err)
		return err
	}
	defer file.Close()

	for _, d := range data {
		_, writeErr := file.WriteString(fmt.Sprintf("%s\n", d))

		if writeErr != nil {
			golog.Fatal(writeErr)
			return writeErr
		}
	}

	return nil
}

//WriteBytesFile write bytes to a file to basepath with filename and the given format
func WriteBytesFile(landscapeDir, basePath, fileName string, data []byte) error {

	dir := fmt.Sprintf("%s/%s", landscapeDir, basePath)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := CreateDirectory(dir); err != nil {
			return err
		}
	}

	path := fmt.Sprintf("%s/%s", dir, fileName)

	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)

	if err != nil {
		golog.Fatal(err)
		return err
	}
	defer file.Close()

	_, writeErr := file.Write(data)

	if writeErr != nil {
		golog.Fatal(writeErr)
		return writeErr
	}

	return nil
}

//WriteFile write a file to basepath with filename and the given format
func WriteFile(landscapeDir, basePath, fileName, format string, resource interface{}) error {
	if format == "YAML" {
		return WriteYAML(landscapeDir, basePath, fileName, resource)
	}

	return WriteJSON(landscapeDir, basePath, fileName, resource)
}

//WriteJSON write JSON to a file to basepath with filename
func WriteJSON(landscapeDir, basePath, fileName string, resource interface{}) error {

	y, err := json.Marshal(resource)

	if err != nil {
		return err
	}

	return WriteBytesFile(landscapeDir, basePath, fileName, y)
}

//WriteYAML write YAMLto a file to basepath with filename
func WriteYAML(landscapeDir, basePath, fileName string, resource interface{}) error {

	y, err := ToYaml(resource)

	if err != nil {
		return err
	}

	return WriteBytesFile(landscapeDir, basePath, fileName, y)
}
