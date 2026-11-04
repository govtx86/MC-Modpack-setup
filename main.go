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

type Mod struct {
	id          string
	slug        string
	url         string
	status      string
	environment string
}

func main() {
	var gameversion, loader, filename string
	flag.StringVar(&gameversion, "version", "26.2", "game version")
	flag.StringVar(&loader, "loader", "fabric", "modloader")
	flag.StringVar(&filename, "filename", "modlist.txt", "name of file containing modlist")

	flag.Parse()

	modlist, err := ReadModList(filename)
	unique_deps := make(map[string]bool)
	var versionlist []VersionResponse
	var deplist []VersionResponse
	if err != nil {
		fmt.Printf("Failed to read file: %s\n", err)
		return
	}
	for _, mod := range modlist {
		fmt.Printf("Slug: %s, %s\n", mod.slug, mod.status)
		version, err := GetVersionInfo(mod.slug, loader, gameversion)
		if err != nil {
			fmt.Printf("Failed to get version info: %s\n", err)
			return
		}
		versionlist = append(versionlist, version[0])
		// fmt.Printf("%s ", version[0].Files[0].URL)
		fmt.Printf("Version info: %s, %s\n", mod.slug, version[0].Files[0].URL)
		for _, dep := range version[0].Dependencies {
			if dep.DependencyType == "required" && !unique_deps[dep.ProjectID] {
				fmt.Printf("Dependency: %s\n", dep.ProjectID)
				depversion, err := GetVersionInfo(dep.ProjectID, loader, gameversion)
				if err != nil {
					fmt.Printf("Failed to get version info: %s\n", err)
					return
				}
				deplist = append(deplist, depversion[0])
				unique_deps[dep.ProjectID] = true
				fmt.Printf("Dependency version info: %s, %s\n", mod.slug, depversion[0].Files[0].URL)
			}
		}
	}
}

func ReadModList(filename string) ([]Mod, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var mods []Mod
	unique_mods := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() != "" {			
			slug := GetSlug(scanner.Text())
			if !unique_mods[slug] {
				mods = append(mods, Mod{
					slug:   slug,
					status: "pending",
				})
				unique_mods[slug] = true
			}
		}
	}
	return mods, scanner.Err()
}

func GetSlug(url string) string {
	slug := strings.Split(url, "/")[4]
	if i := strings.Index(slug, "?"); i != -1 {
		slug = slug[:i]
	}
	return slug
}

type VersionResponse struct {
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
		URL      string `json:"url"`
		Filename string `json:"filename"`
		Primary  bool   `json:"primary"`
		Size     int    `json:"size"`
	} `json:"files"`
	Dependencies []struct {
		ProjectID      string `json:"project_id"`
		DependencyType string `json:"dependency_type"`
	} `json:"dependencies"`
}

func GetVersionInfo(slug, loader, gameversion string) ([]VersionResponse, error) {
	params := url.Values{}
	params.Add("loaders", fmt.Sprintf("[\"%s\"]", loader))
	params.Add("game_versions", fmt.Sprintf("[\"%s\"]", gameversion))
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.modrinth.com/v2/project/%s/version?", slug)+params.Encode(), nil)
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
	var response []VersionResponse
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response, nil
}
