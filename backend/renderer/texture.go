package renderer

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fogleman/fauxgl"
)

var (
	bsurl         = "https://api.buildoverse.com"
	httpClient    = &http.Client{Timeout: 5 * time.Second}
	downloadMutex sync.Mutex
	fileLocks     = make(map[string]*sync.Mutex)
)

type CompositeTexture struct {
	Layers []fauxgl.Texture
	Color  fauxgl.Color
}

func (t CompositeTexture) Sample(u, v float64) fauxgl.Color {
	return t.blendLayers(u, v, false)
}

func (t CompositeTexture) BilinearSample(u, v float64) fauxgl.Color {
	return t.blendLayers(u, v, true)
}

func (t CompositeTexture) blendLayers(u, v float64, bilinear bool) fauxgl.Color {
	final := t.Color
	final.A = 1.0
	for _, l := range t.Layers {
		if l == nil {
			continue
		}
		var c fauxgl.Color
		if bilinear {
			c = l.BilinearSample(u, v)
		} else {
			c = l.Sample(u, v)
		}
		invA := 1.0 - c.A
		final.R = c.R*c.A + final.R*invA
		final.G = c.G*c.A + final.G*invA
		final.B = c.B*c.A + final.B*invA
	}
	return final
}

func getFileLock(path string) *sync.Mutex {
	downloadMutex.Lock()
	defer downloadMutex.Unlock()
	if lock, exists := fileLocks[path]; exists {
		return lock
	}
	lock := &sync.Mutex{}
	fileLocks[path] = lock
	return lock
}

func assetURL(itemType string, id int, ext string) string {
	return fmt.Sprintf("%s/assets/uploads/shop/%s/%d.%s?v=%d", bsurl, itemType, id, ext, time.Now().UnixNano())
}

func downloadFile(url string, dest string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status code error: %d", resp.StatusCode)
	}
	_ = os.MkdirAll(filepath.Dir(dest), 0755)
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func fetchTexture(itemType string, id int, fallback string) (fauxgl.Texture, error) {
	if id <= 0 {
		if fallback != "" {
			return fauxgl.LoadTexture(fallback)
		}
		return nil, fmt.Errorf("invalid id")
	}
	cleanType := filepath.Base(strings.TrimSpace(itemType))

	localCandidates := []string{
		filepath.Join("assets", "uploads", "shop", cleanType, fmt.Sprintf("%d.png", id)),
		filepath.Join("assets", "shop", cleanType, fmt.Sprintf("%d.png", id)),
		filepath.Join("static", "uploads", "shop", cleanType, fmt.Sprintf("%d.png", id)),
		filepath.Join("uploads", "shop", cleanType, fmt.Sprintf("%d.png", id)),
	}
	if cleanType == "faces" {
		localCandidates = append(localCandidates, filepath.Join(FacePath, fmt.Sprintf("%d.png", id)))
	}

	for _, localPath := range localCandidates {
		if _, err := os.Stat(localPath); err == nil {
			if tex, err := fauxgl.LoadTexture(localPath); err == nil {
				return tex, nil
			}
		}
	}

	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s_%d.png", cleanType, id))

	lock := getFileLock(tmpPath)
	lock.Lock()
	defer lock.Unlock()

	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		if err := downloadFile(assetURL(cleanType, id, "png"), tmpPath); err != nil {
			os.Remove(tmpPath)
			if fallback != "" {
				return fauxgl.LoadTexture(fallback)
			}
			return nil, err
		}
	}

	tex, err := fauxgl.LoadTexture(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		if fallback != "" {
			return fauxgl.LoadTexture(fallback)
		}
	}
	return tex, err
}

func fetchMesh(itemType string, id int) (*fauxgl.Mesh, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid id")
	}
	cleanType := filepath.Base(strings.TrimSpace(itemType))

	localCandidates := []string{
		filepath.Join("assets", "uploads", "shop", cleanType, fmt.Sprintf("%d.obj", id)),
		filepath.Join("assets", "shop", cleanType, fmt.Sprintf("%d.obj", id)),
		filepath.Join("static", "uploads", "shop", cleanType, fmt.Sprintf("%d.obj", id)),
		filepath.Join("uploads", "shop", cleanType, fmt.Sprintf("%d.obj", id)),
	}

	for _, localPath := range localCandidates {
		if _, err := os.Stat(localPath); err == nil {
			if mesh, err := fauxgl.LoadOBJ(localPath); err == nil && mesh != nil && len(mesh.Triangles) > 0 {
				return mesh, nil
			}
		}
	}

	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s_%d.obj", cleanType, id))

	lock := getFileLock(tmpPath)
	lock.Lock()
	defer lock.Unlock()

	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		if err := downloadFile(assetURL(cleanType, id, "obj"), tmpPath); err != nil {
			os.Remove(tmpPath)
			return nil, err
		}
	}

	mesh, err := fauxgl.LoadOBJ(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		log.Printf("failed loading mesh %s: %v", cleanType, err)
	}
	return mesh, err
}
