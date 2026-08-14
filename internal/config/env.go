package config

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// LoadDotEnv reads KEY=VALUE pairs from path into the process environment.
// Existing variables win so real env always overrides the file.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	for k, v := range ParseDotEnv(f) {
		if _, set := os.LookupEnv(k); !set {
			os.Setenv(k, v)
		}
	}
	return nil
}

// ParseDotEnv parses simple KEY=VALUE lines, skipping comments and blanks.
func ParseDotEnv(r io.Reader) map[string]string {
	vars := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		vars[key] = value
	}
	return vars
}
