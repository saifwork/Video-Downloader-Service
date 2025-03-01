package utils

import (
	"net/url"
	"os"
	"strings"
)

// Check if file exists before sending
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "None"
	}
	return strings.Join(items, ", ")
}
