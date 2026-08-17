package server

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type BrowseEntry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"isDir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Ext     string    `json:"ext"`
	IsVideo bool      `json:"isVideo"`
}

type Bookmark struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

type BrowseResponse struct {
	Path      string        `json:"path"`
	Parent    string        `json:"parent,omitempty"`
	Entries   []BrowseEntry `json:"entries"`
	Bookmarks []Bookmark    `json:"bookmarks"`
}

var videoExts = map[string]bool{
	".mp4": true, ".mov": true, ".mkv": true, ".m4v": true, ".avi": true, ".webm": true,
}

func bookmarks() []Bookmark {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []Bookmark{
		{Label: "Home", Path: home},
		{Label: "Downloads", Path: filepath.Join(home, "Downloads")},
		{Label: "Movies", Path: filepath.Join(home, "Movies")},
		{Label: "Desktop", Path: filepath.Join(home, "Desktop")},
	}
}

func handleBrowse(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	if p == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		p = home
	}
	if !filepath.IsAbs(p) {
		writeJSONError(w, http.StatusBadRequest, "path must be an absolute path")
		return
	}
	p = filepath.Clean(p)

	entries, err := os.ReadDir(p)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			writeJSONError(w, http.StatusNotFound, "Path does not exist")
		case os.IsPermission(err):
			// macOS TCC (privacy permissions) commonly blocks non-Terminal-launched
			// processes from reading ~/Desktop, ~/Downloads, ~/Movies, etc.
			// A bare 403 here would leave the user with no idea why -- spell
			// out the fix.
			writeJSONError(w, http.StatusForbidden,
				"Permission denied. Please grant Full Disk Access to Terminal (or whisper-gui) in System Settings → Privacy & Security, then try again")
		default:
			writeJSONError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	out := make([]BrowseEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		out = append(out, BrowseEntry{
			Name:    e.Name(),
			Path:    filepath.Join(p, e.Name()),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Ext:     ext,
			IsVideo: videoExts[ext],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})

	resp := BrowseResponse{Path: p, Entries: out, Bookmarks: bookmarks()}
	if parent := filepath.Dir(p); parent != p {
		resp.Parent = parent
	}
	writeJSON(w, http.StatusOK, resp)
}
