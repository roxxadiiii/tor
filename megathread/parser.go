package main

import (
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
)

// Hack to map Description to bubbles/list correctly while keeping the raw string
type BookmarkItem struct {
	Name    string
	URL     string
	DescStr string
}

func (b BookmarkItem) Title() string       { return b.Name }
func (b BookmarkItem) Description() string { 
	if b.DescStr != "" {
		return b.DescStr
	}
	return b.URL
}
func (b BookmarkItem) FilterValue() string { return b.Name }

type Folder struct {
	Name      string
	Bookmarks []BookmarkItem
}

func (f Folder) Title() string       { return f.Name }
func (f Folder) Description() string { return "Folder containing " + string(rune(len(f.Bookmarks)+48)) + " items" } // Simplified int-to-str for small counts
func (f Folder) FilterValue() string { return f.Name }

func ParseMegathread(filepath string) ([]Folder, error) {
	file, err := os.Open(filepath)
	if err != nil { return nil, err }
	defer file.Close()
	raw, _ := io.ReadAll(file)
	
	var data struct {
		Data struct { ContentMd string `json:"content_md"` } `json:"data"`
	}
	if err := json.Unmarshal(raw, &data); err != nil { return nil, err }
	
	lines := strings.Split(data.Data.ContentMd, "\n")
	
	var folders []Folder
	var currentFolder *Folder
	var currentBookmark *BookmarkItem

	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\)]+)\)`)
	headingRegex := regexp.MustCompile(`^(?:#|##|###)\s+(.*)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "---" || line == "&nbsp;" { continue }
		if strings.HasPrefix(line, "![]") { continue }
		
		if match := headingRegex.FindStringSubmatch(line); len(match) > 1 {
			titleText := match[1]
			if links := linkRegex.FindAllStringSubmatch(titleText, -1); len(links) > 0 {
				l := links[0]
				cleanTitle := strings.ReplaceAll(l[1], "**", "")
				if currentFolder == nil {
					folders = append(folders, Folder{Name: "General"})
					currentFolder = &folders[len(folders)-1]
				}
				bm := BookmarkItem{ Name: cleanTitle, URL: l[2], DescStr: "" }
				currentFolder.Bookmarks = append(currentFolder.Bookmarks, bm)
				currentBookmark = &currentFolder.Bookmarks[len(currentFolder.Bookmarks)-1]
			} else {
				cleanTitle := strings.ReplaceAll(titleText, "**", "")
				cleanTitle = strings.ReplaceAll(cleanTitle, "➔ ", "")
				folders = append(folders, Folder{Name: cleanTitle})
				currentFolder = &folders[len(folders)-1]
				currentBookmark = nil
			}
			continue
		}
		
		if links := linkRegex.FindAllStringSubmatch(line, -1); len(links) > 0 {
			if currentFolder == nil {
				folders = append(folders, Folder{Name: "General"})
				currentFolder = &folders[len(folders)-1]
			}
			cleanLine := strings.ReplaceAll(line, "**", "")
			cleanLine = strings.ReplaceAll(cleanLine, "`", "")
			
			for _, l := range links {
				cleanTitle := strings.ReplaceAll(l[1], "**", "")
				bm := BookmarkItem{ Name: cleanTitle, URL: l[2], DescStr: cleanLine }
				currentFolder.Bookmarks = append(currentFolder.Bookmarks, bm)
				currentBookmark = &currentFolder.Bookmarks[len(currentFolder.Bookmarks)-1]
			}
			continue
		}
		
		if currentBookmark != nil {
			cleanLine := strings.ReplaceAll(line, "**", "")
			cleanLine = strings.ReplaceAll(cleanLine, "> ", "")
			if cleanLine != "" {
				if currentBookmark.DescStr != "" && !strings.Contains(currentBookmark.DescStr, cleanLine) {
					currentBookmark.DescStr += " - " + cleanLine
				} else if currentBookmark.DescStr == "" {
					currentBookmark.DescStr = cleanLine
				}
			}
		}
	}
	
	var final []Folder
	for _, f := range folders {
		if len(f.Bookmarks) > 0 {
			// update counts cleanly
			fName := f.Name
			final = append(final, Folder{Name: fName, Bookmarks: f.Bookmarks})
		}
	}
	return final, nil
}
