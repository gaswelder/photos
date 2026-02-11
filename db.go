package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"time"
)

// album is a struct describing a local album directory.
type album struct {
	Path         string
	ReverseOrder bool
	PathAsName   bool

	allEntries  []entry
	entriesTime time.Time
}

// entry is a single item in an album.
// An entry may be just one image file with optional description,
// or a group of images grouped together in a subdirectory.
type entry struct {
	path   string
	Name   string
	Desc   string
	Images []string
}

// Don't want to expose actual file names, so instead they are hashed
// and mappings are kept in this lookup.
// This is fine as the albums are not expected to be huge.
// The mapping is global across all albums.
var hashlock sync.Mutex
var pathToHash map[string]string
var hashToPath map[string]string

func init() {
	pathToHash = make(map[string]string)
	hashToPath = make(map[string]string)
}

// imageID returns a unique ID for the image at the given path.
func imageID(path string) string {
	hashlock.Lock()
	defer hashlock.Unlock()
	h, ok := pathToHash[path]
	if ok {
		return h
	}
	h = fmt.Sprintf("%x", sha1.Sum([]byte(path)))
	pathToHash[path] = h
	hashToPath[h] = path
	return h
}

// imagePath returns the file path for the image with the given id.
func imagePath(id string) string {
	return hashToPath[id]
}

func (a *album) reload() error {
	dir, err := os.ReadDir(a.Path)
	if err != nil {
		return err
	}
	if a.ReverseOrder {
		slices.Reverse(dir)
	}
	var entries []entry
	for _, e := range dir {
		m, err := a.loadEntry(e.Name(), e.IsDir())
		if err != nil {
			continue
		}
		entries = append(entries, m)
	}
	a.allEntries = entries
	a.entriesTime = time.Now()
	return nil
}

// entries returns all entries from the album.
func (a *album) entries(filter string) ([]entry, error) {
	var entries []entry
	filter = strings.ToLower(filter)
	for _, e := range a.allEntries {
		if !strings.Contains(strings.ToLower(e.Name), filter) {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (a *album) loadEntry(name string, isDir bool) (entry, error) {
	var m entry

	if !isDir {
		imgpath := a.Path + "/" + name
		ext := path.Ext(imgpath)
		if !isImageExt(ext) {
			return m, fmt.Errorf("unknown file: %s", name)
		}
		descPath := strings.Replace(imgpath, ext, ".txt", 1)
		desc, _ := getText(descPath)
		m.path = imgpath
		m.Desc = desc
		m.Images = append(m.Images, imageID(imgpath))
		if a.PathAsName {
			m.Name = path.Base(imgpath)
			m.Name = strings.Replace(m.Name, path.Ext(m.Name), "", 1)
		}
		return m, nil
	}

	dirPath := a.Path + "/" + name
	m.path = dirPath
	dir, err := os.ReadDir(dirPath)
	if err != nil {
		return m, nil
	}

	if a.PathAsName {
		m.Name = path.Base(dirPath)
	}
	for _, item := range dir {
		itemPath := dirPath + "/" + item.Name()
		if isImageExt(path.Ext(item.Name())) {
			m.Images = append(m.Images, imageID(itemPath))
			continue
		}

		switch path.Ext(item.Name()) {
		case ".txt":
			desc, err := getText(itemPath)
			if err != nil {
				return m, err
			}
			m.Desc += desc
		default:
			fmt.Println("unknown item", itemPath)
		}
	}
	return m, nil
}

func getText(itemPath string) (string, error) {
	data, err := os.ReadFile(itemPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
