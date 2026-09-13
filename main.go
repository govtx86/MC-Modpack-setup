package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type ModList []struct {
	slug   string
	status string
}

func main() {
	var gameversion, loader, filename string
	flag.StringVar(&gameversion, "version", "26.2", "game version")
	flag.StringVar(&loader, "loader", "fabric", "modloader")
	flag.StringVar(&filename, "filename", "modlist.txt", "name of file containing modlist")

	flag.Parse()

	modlist, err := ReadModList(filename)
	var versionlist []VersionResponse
	// var deplist ModList
	if err != nil {
		fmt.Printf("Failed to read file: %s\n", err)
		return
	}
	for _, mod := range modlist {
		fmt.Printf("Slug: %s, %s\n", mod.slug, mod.status)
		version, err := GetVersionInfo(mod.slug, loader, gameversion)
		if err != nil {
			fmt.Printf("Failed to get version info:%s\n", err)
			return
		}
		versionlist = append(versionlist, version)
		fmt.Printf("Version info: %s, %s\n", mod.slug, version[0].Files[0].URL)
	}
}

func ReadModList(filename string) (ModList, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var mods ModList
	unique := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		slug := strings.Split(scanner.Text(), "/")[4]
		if !unique[slug] {
			mods = append(mods, struct {
				slug   string
				status string
			}{slug, "pending"})
			unique[slug] = true
		}
	}
	return mods, scanner.Err()
}

type VersionResponse []struct {
	GameVersions  []string  `json:"game_versions"`
	Loaders       []string  `json:"loaders"`
	Environment   string    `json:"environment"`
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	DatePublished time.Time `json:"date_published"`
	Files         []struct {
		ID     string `json:"id"`
		Hashes struct {
			Sha1   string `json:"sha1"`
			Sha512 string `json:"sha512"`
		} `json:"hashes"`
		URL      string      `json:"url"`
		Filename string      `json:"filename"`
		Primary  bool        `json:"primary"`
		Size     int         `json:"size"`
		FileType interface{} `json:"file_type"`
	} `json:"files"`
	Dependencies []struct {
		VersionID      interface{} `json:"version_id"`
		ProjectID      string      `json:"project_id"`
		FileName       interface{} `json:"file_name"`
		DependencyType string      `json:"dependency_type"`
	} `json:"dependencies"`
}

func GetVersionInfo(slug, loader, gameversion string) (VersionResponse, error) {
	params := url.Values{}
	params.Add("loaders", fmt.Sprintf("[%s]", loader))
	params.Add("game_versions", fmt.Sprintf("[%s]", gameversion))
	fmt.Println(params.Encode())
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.modrinth.com/v2/project/%s/version?", slug) + params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Modpack_setup_tool")
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var response VersionResponse
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response, nil
}
