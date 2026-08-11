package discovery

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"leigod-auto-pause/internal/config"
)

var vdfValue = regexp.MustCompile(`"([^"]+)"\s+"([^"]*)"`)

func InstalledGames() []config.Game {
	games := append(discoverSteam(), discoverEpic()...)
	seen := map[string]bool{}
	result := make([]config.Game, 0, len(games))
	for _, game := range games {
		key := strings.ToLower(game.Executable)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, game)
	}
	sort.Slice(result, func(i, j int) bool { return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name) })
	return result
}

func discoverSteam() []config.Game {
	roots := []string{}
	for _, base := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles")} {
		if base != "" {
			roots = append(roots, filepath.Join(base, "Steam"))
		}
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		roots = append(roots, filepath.Join(local, "Steam"))
	}
	libraries := map[string]bool{}
	for _, root := range roots {
		if _, err := os.Stat(filepath.Join(root, "steamapps")); err == nil {
			libraries[root] = true
		}
		data, err := os.ReadFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"))
		if err != nil {
			continue
		}
		for _, match := range vdfValue.FindAllStringSubmatch(string(data), -1) {
			if strings.EqualFold(match[1], "path") {
				libraries[strings.ReplaceAll(match[2], `\\`, `\`)] = true
			}
		}
	}
	var games []config.Game
	for library := range libraries {
		manifests, _ := filepath.Glob(filepath.Join(library, "steamapps", "appmanifest_*.acf"))
		for _, manifest := range manifests {
			data, err := os.ReadFile(manifest)
			if err != nil {
				continue
			}
			values := map[string]string{}
			for _, match := range vdfValue.FindAllStringSubmatch(string(data), -1) {
				values[strings.ToLower(match[1])] = match[2]
			}
			install := filepath.Join(library, "steamapps", "common", values["installdir"])
			exe := findLikelyExecutable(install)
			if exe != "" {
				games = append(games, config.Game{Name: values["name"], Executable: filepath.Base(exe), Path: exe, Source: "Steam", Enabled: true})
			}
		}
	}
	return games
}

type epicManifest struct {
	DisplayName      string `json:"DisplayName"`
	InstallLocation  string `json:"InstallLocation"`
	LaunchExecutable string `json:"LaunchExecutable"`
}

func discoverEpic() []config.Game {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		return nil
	}
	files, _ := filepath.Glob(filepath.Join(programData, "Epic", "EpicGamesLauncher", "Data", "Manifests", "*.item"))
	var games []config.Game
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var item epicManifest
		if json.Unmarshal(data, &item) != nil || item.LaunchExecutable == "" {
			continue
		}
		path := item.LaunchExecutable
		if !filepath.IsAbs(path) {
			path = filepath.Join(item.InstallLocation, path)
		}
		games = append(games, config.Game{Name: item.DisplayName, Executable: filepath.Base(path), Path: path, Source: "Epic", Enabled: true})
	}
	return games
}

func findLikelyExecutable(root string) string {
	var best string
	var bestSize int64
	visited := 0
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		visited++
		if visited > 12000 {
			return fs.SkipAll
		}
		lower := strings.ToLower(path)
		if entry.IsDir() {
			if lower != strings.ToLower(root) && (strings.Contains(lower, "redist") || strings.Contains(lower, "installer") || strings.Contains(lower, "crashreport")) {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(lower) != ".exe" || strings.Contains(lower, "unins") || strings.Contains(lower, "launcher") || strings.Contains(lower, "crash") {
			return nil
		}
		info, err := entry.Info()
		if err == nil && info.Size() > bestSize {
			best, bestSize = path, info.Size()
		}
		return nil
	})
	return best
}
