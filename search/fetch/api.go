package fetch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Torrent struct {
	Name       string
	Magnet     string
	Seeders    int
	Leechers   int
	Size       string
	SizeRaw    int64 // For precise sorting
	UploadDate string
	OriginSite string
}

// APIBay Structs
type apibayResult struct {
	Name     string `json:"name"`
	InfoHash string `json:"info_hash"`
	Seeders  string `json:"seeders"`
	Leechers string `json:"leechers"`
	Size     string `json:"size"`
	Added    string `json:"added"`
}

// YTS Structs
type ytsResponse struct {
	Data struct {
		Movies []struct {
			Title        string `json:"title_long"`
			DateUploaded string `json:"date_uploaded"`
			Torrents     []struct {
				Hash  string `json:"hash"`
				Seeds int    `json:"seeds"`
				Peers int    `json:"peers"`
				Size  string `json:"size"`
				SizeB int64  `json:"size_bytes"`
			} `json:"torrents"`
		} `json:"movies"`
	} `json:"data"`
}

// EZTV Structs
type eztvResponse struct {
	Torrents []struct {
		Title    string `json:"title"`
		Magnet   string `json:"magnet_url"`
		Seeds    int    `json:"seeds"`
		Peers    int    `json:"peers"`
		SizeByte string `json:"size_bytes"`
		Date     int64  `json:"date_released_unix"`
	} `json:"torrents"`
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FetchConcurrently multiplexes the search
func FetchConcurrently(query string) ([]Torrent, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []Torrent

	// 1. YTS Fetcher
	wg.Add(1)
	go func() {
		defer wg.Done()
		encoded := url.QueryEscape(query)
		u := "https://yts.mx/api/v2/list_movies.json?query_term=" + encoded + "&limit=50"
		resp, err := http.Get(u)
		if err != nil { return }
		defer resp.Body.Close()

		var yts ytsResponse
		if err := json.NewDecoder(resp.Body).Decode(&yts); err == nil {
			for _, m := range yts.Data.Movies {
				for _, t := range m.Torrents {
					mu.Lock()
					results = append(results, Torrent{
						Name:       fmt.Sprintf("%s (%s)", m.Title, t.Size),
						Magnet:     fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", t.Hash, url.QueryEscape(m.Title)),
						Seeders:    t.Seeds,
						Leechers:   t.Peers,
						Size:       t.Size,
						SizeRaw:    t.SizeB,
						UploadDate: m.DateUploaded,
						OriginSite: "YTS",
					})
					mu.Unlock()
				}
			}
		}
	}()

	// 2. APIBay Fetcher
	wg.Add(1)
	go func() {
		defer wg.Done()
		encoded := url.QueryEscape(query)
		// Requesting highest capability
		u := "https://apibay.org/q.php?q=" + encoded
		resp, err := http.Get(u)
		if err != nil { return }
		defer resp.Body.Close()

		var bay []apibayResult
		if err := json.NewDecoder(resp.Body).Decode(&bay); err == nil {
			for _, r := range bay {
				if r.InfoHash == "0000000000000000000000000000000000000000" || r.Name == "No results returned" || r.InfoHash == "" {
					continue
				}

				seeds, _ := strconv.Atoi(r.Seeders)
				leechers, _ := strconv.Atoi(r.Leechers)
				addedInt, _ := strconv.ParseInt(r.Added, 10, 64)
				sizeInt, _ := strconv.ParseInt(r.Size, 10, 64)

				mu.Lock()
				results = append(results, Torrent{
					Name:       r.Name,
					Magnet:     fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", r.InfoHash, url.QueryEscape(r.Name)),
					Seeders:    seeds,
					Leechers:   leechers,
					Size:       formatSize(sizeInt),
					SizeRaw:    sizeInt,
					UploadDate: time.Unix(addedInt, 0).Format("2006-01-02"),
					OriginSite: "PirateBay",
				})
				mu.Unlock()
			}
		}
	}()

	// 3. EZTV Fetcher (Only if query resembles an IMDB ID)
	imdbRegex := regexp.MustCompile(`^(?:tt)?(\d{7,8})$`)
	if match := imdbRegex.FindStringSubmatch(query); len(match) > 1 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			imdbID := match[1] // The numeric part
			u := "https://eztv.re/api/get-torrents?limit=100&imdb_id=" + imdbID
			resp, err := http.Get(u)
			if err != nil { return }
			defer resp.Body.Close()

			var ez eztvResponse
			if err := json.NewDecoder(resp.Body).Decode(&ez); err == nil {
				for _, t := range ez.Torrents {
					sizeInt, _ := strconv.ParseInt(t.SizeByte, 10, 64)
					mu.Lock()
					results = append(results, Torrent{
						Name:       t.Title,
						Magnet:     t.Magnet,
						Seeders:    t.Seeds,
						Leechers:   t.Peers,
						Size:       formatSize(sizeInt),
						SizeRaw:    sizeInt,
						UploadDate: time.Unix(t.Date, 0).Format("2006-01-02"),
						OriginSite: "EZTV",
					})
					mu.Unlock()
				}
			}
		}()
	}

	// 4. Academic Torrents Scraper Workaround
	wg.Add(1)
	go func() {
		defer wg.Done()
		u := "https://academictorrents.com/browse.php?search=" + url.QueryEscape(query)
		resp, err := http.Get(u)
		if err != nil || resp.StatusCode != 200 { return }
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil { return }

		// Searching global table rows
		doc.Find("tr").Each(func(i int, s *goquery.Selection) {
			title := strings.TrimSpace(s.Find("td").Eq(1).Find("a").First().Text())
			magnet, hasMagnet := s.Find("a[href^='magnet:?']").Attr("href")
			if !hasMagnet || title == "" { return }

			sizeStr := strings.TrimSpace(s.Find("td").Eq(2).Text())
			
			mu.Lock()
			results = append(results, Torrent{
				Name:       title,
				Magnet:     magnet,
				Seeders:    0, // AT hides seeds from the raw unstructured list without deeper parsing
				Leechers:   0,
				Size:       sizeStr,
				SizeRaw:    0,
				UploadDate: "Unknown",
				OriginSite: "AcadTorrents",
			})
			mu.Unlock()
		})
	}()

	wg.Wait()
	return results, nil
}
