package fetch

import (
	"fmt"
	"io"
	"net/http"
)

var TrackerURLs = map[string]string{
	"All":   "https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all.txt",
	"HTTP":  "https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_http.txt",
	"HTTPS": "https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_https.txt",
	"IP":    "https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_ip.txt",
	"Best":  "https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt",
}

func FetchTrackers(kind string) (string, error) {
	url, ok := TrackerURLs[kind]
	if !ok {
		return "", fmt.Errorf("unknown tracker kind: %s", kind)
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch trackers, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
