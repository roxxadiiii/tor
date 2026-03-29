package core

import (
	"os/exec"
	"strings"
)

func IsYoutube(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func GetYTDLStream(url string) ([]string, error) {
	// Extracts direct video/audio unified payload stream url
	cmd := exec.Command("yt-dlp", "-f", "best", "-g", url)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var valid []string
	for _, l := range lines {
		if l != "" {
			valid = append(valid, l)
		}
	}
	return valid, nil
}
