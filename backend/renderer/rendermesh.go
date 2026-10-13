package renderer

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/fogleman/fauxgl"
)

type gltfJSON struct {
	Accessors []struct {
		BufferView    *int   `json:"bufferView"`
		ByteOffset    int    `json:"byteOffset"`
		ComponentType int    `json:"componentType"`
		Count         int    `json:"count"`
		Type          string `json:"type"`
	} `json:"accessors"`
	BufferViews []struct {
		Buffer     int `json:"buffer"`
		ByteOffset int `json:"byteOffset"`
		ByteLength int `json:"byteLength"`
		ByteStride int `json:"byteStride"`
	} `json:"bufferViews"`
	Meshes []struct {
		Primitives []struct {
			Attributes map[string]int `json:"attributes"`
			Indices    *int           `json:"indices"`
			Mode       *int           `json:"mode"`
		} `json:"primitives"`
	} `json:"meshes"`
	Images []struct {
		BufferView *int `json:"bufferView"`
	} `json:"images"`
}

func readVec3Accessor(doc gltfJSON, bin []byte, accIdx int) []fauxgl.Vector {
	acc := doc.Accessors[accIdx]
	if acc.BufferView == nil || *acc.BufferView < 0 || *acc.BufferView >= len(doc.BufferViews) {
		return nil
	}
	bv := doc.BufferViews[*acc.BufferView]
	stride := bv.ByteStride
	if stride == 0 {
		stride = 12
	}
	start := bv.ByteOffset + acc.ByteOffset
	result := make([]fauxgl.Vector, acc.Count)
	for i := 0; i < acc.Count; i++ {
		offset := start + i*stride
		if offset+12 > len(bin) {
			break
		}
		x := math.Float32frombits(binary.LittleEndian.Uint32(bin[offset : offset+4]))
		y := math.Float32frombits(binary.LittleEndian.Uint32(bin[offset+4 : offset+8]))
		z := math.Float32frombits(binary.LittleEndian.Uint32(bin[offset+8 : offset+12]))
		result[i] = fauxgl.Vector{X: float64(x), Y: float64(y), Z: float64(z)}
	}
	return result
}

func readVec2Accessor(doc gltfJSON, bin []byte, accIdx int) []fauxgl.Vector {
	acc := doc.Accessors[accIdx]
	if acc.BufferView == nil || *acc.BufferView < 0 || *acc.BufferView >= len(doc.BufferViews) {
		return nil
	}
	bv := doc.BufferViews[*acc.BufferView]
	stride := bv.ByteStride
	if stride == 0 {
		stride = 8
	}
	start := bv.ByteOffset + acc.ByteOffset
	result := make([]fauxgl.Vector, acc.Count)
	for i := 0; i < acc.Count; i++ {
		offset := start + i*stride
		if offset+8 > len(bin) {
			break
		}
		u := math.Float32frombits(binary.LittleEndian.Uint32(bin[offset : offset+4]))
		v := math.Float32frombits(binary.LittleEndian.Uint32(bin[offset+4 : offset+8]))
		result[i] = fauxgl.Vector{X: float64(u), Y: float64(v), Z: 0}
	}
	return result
}

func readIndicesAccessor(doc gltfJSON, bin []byte, accIdx int) []int {
	acc := doc.Accessors[accIdx]
	if acc.BufferView == nil || *acc.BufferView < 0 || *acc.BufferView >= len(doc.BufferViews) {
		return nil
	}
	bv := doc.BufferViews[*acc.BufferView]
	start := bv.ByteOffset + acc.ByteOffset
	result := make([]int, acc.Count)

	switch acc.ComponentType {
	case 5121:
		stride := bv.ByteStride
		if stride == 0 {
			stride = 1
		}
		for i := 0; i < acc.Count; i++ {
			offset := start + i*stride
			if offset >= len(bin) {
				break
			}
			result[i] = int(bin[offset])
		}
	case 5123:
		stride := bv.ByteStride
		if stride == 0 {
			stride = 2
		}
		for i := 0; i < acc.Count; i++ {
			offset := start + i*stride
			if offset+2 > len(bin) {
				break
			}
			result[i] = int(binary.LittleEndian.Uint16(bin[offset : offset+2]))
		}
	case 5125:
		stride := bv.ByteStride
		if stride == 0 {
			stride = 4
		}
		for i := 0; i < acc.Count; i++ {
			offset := start + i*stride
			if offset+4 > len(bin) {
				break
			}
			result[i] = int(binary.LittleEndian.Uint32(bin[offset : offset+4]))
		}
	}

	return result
}

func loadGLB(path string) (*fauxgl.Mesh, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var header struct {
		Magic   uint32
		Version uint32
		Length  uint32
	}
	if err := binary.Read(file, binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	if header.Magic != 0x46546C67 {
		return nil, errors.New("invalid glb magic")
	}

	var jsonBytes []byte
	var binBytes []byte

	for {
		var chunkHeader struct {
			Length uint32
			Type   uint32
		}
		if err := binary.Read(file, binary.LittleEndian, &chunkHeader); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, err
		}

		chunkData := make([]byte, chunkHeader.Length)
		if _, err := io.ReadFull(file, chunkData); err != nil {
			return nil, err
		}

		if chunkHeader.Type == 0x4E4F534A && jsonBytes == nil {
			jsonBytes = chunkData
		} else if chunkHeader.Type == 0x004E4942 && binBytes == nil {
			binBytes = chunkData
		}
	}

	if len(jsonBytes) == 0 {
		return nil, errors.New("missing json chunk in glb")
	}

	var doc gltfJSON
	if err := json.Unmarshal(jsonBytes, &doc); err != nil {
		return nil, err
	}

	var triangles []*fauxgl.Triangle

	for _, mesh := range doc.Meshes {
		for _, prim := range mesh.Primitives {
			posAccIdx, hasPos := prim.Attributes["POSITION"]
			if !hasPos || posAccIdx < 0 || posAccIdx >= len(doc.Accessors) {
				continue
			}

			positions := readVec3Accessor(doc, binBytes, posAccIdx)
			if len(positions) == 0 {
				continue
			}

			var normals []fauxgl.Vector
			if normAccIdx, hasNorm := prim.Attributes["NORMAL"]; hasNorm && normAccIdx >= 0 && normAccIdx < len(doc.Accessors) {
				normals = readVec3Accessor(doc, binBytes, normAccIdx)
			}

			var texcoords []fauxgl.Vector
			if uvAccIdx, hasUV := prim.Attributes["TEXCOORD_0"]; hasUV && uvAccIdx >= 0 && uvAccIdx < len(doc.Accessors) {
				texcoords = readVec2Accessor(doc, binBytes, uvAccIdx)
			}

			if prim.Indices != nil && *prim.Indices >= 0 && *prim.Indices < len(doc.Accessors) {
				indices := readIndicesAccessor(doc, binBytes, *prim.Indices)
				for i := 0; i+2 < len(indices); i += 3 {
					i1, i2, i3 := indices[i], indices[i+1], indices[i+2]
					if i1 < len(positions) && i2 < len(positions) && i3 < len(positions) {
						t := &fauxgl.Triangle{}
						t.V1.Position = positions[i1]
						t.V2.Position = positions[i2]
						t.V3.Position = positions[i3]
						if i1 < len(normals) && i2 < len(normals) && i3 < len(normals) {
							t.V1.Normal = normals[i1]
							t.V2.Normal = normals[i2]
							t.V3.Normal = normals[i3]
						}
						if i1 < len(texcoords) && i2 < len(texcoords) && i3 < len(texcoords) {
							t.V1.Texture = texcoords[i1]
							t.V2.Texture = texcoords[i2]
							t.V3.Texture = texcoords[i3]
						}
						t.FixNormals()
						triangles = append(triangles, t)
					}
				}
			} else {
				for i := 0; i+2 < len(positions); i += 3 {
					t := &fauxgl.Triangle{}
					t.V1.Position = positions[i]
					t.V2.Position = positions[i+1]
					t.V3.Position = positions[i+2]
					if i+2 < len(normals) {
						t.V1.Normal = normals[i]
						t.V2.Normal = normals[i+1]
						t.V3.Normal = normals[i+2]
					}
					if i+2 < len(texcoords) {
						t.V1.Texture = texcoords[i]
						t.V2.Texture = texcoords[i+1]
						t.V3.Texture = texcoords[i+2]
					}
					t.FixNormals()
					triangles = append(triangles, t)
				}
			}
		}
	}

	if len(triangles) == 0 {
		return nil, errors.New("no triangles found in glb")
	}

	return fauxgl.NewTriangleMesh(triangles), nil
}

func extractGLBTexture(path string) fauxgl.Texture {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var header struct {
		Magic   uint32
		Version uint32
		Length  uint32
	}
	if err := binary.Read(file, binary.LittleEndian, &header); err != nil || header.Magic != 0x46546C67 {
		return nil
	}

	var jsonBytes []byte
	var binBytes []byte

	for {
		var chunkHeader struct {
			Length uint32
			Type   uint32
		}
		if err := binary.Read(file, binary.LittleEndian, &chunkHeader); err != nil {
			break
		}

		chunkData := make([]byte, chunkHeader.Length)
		if _, err := io.ReadFull(file, chunkData); err != nil {
			break
		}

		if chunkHeader.Type == 0x4E4F534A && jsonBytes == nil {
			jsonBytes = chunkData
		} else if chunkHeader.Type == 0x004E4942 && binBytes == nil {
			binBytes = chunkData
		}
	}

	if len(jsonBytes) == 0 || len(binBytes) == 0 {
		return nil
	}

	var doc gltfJSON
	if err := json.Unmarshal(jsonBytes, &doc); err != nil || len(doc.Images) == 0 {
		return nil
	}

	if doc.Images[0].BufferView == nil {
		return nil
	}
	bvIdx := *doc.Images[0].BufferView
	if bvIdx < 0 || bvIdx >= len(doc.BufferViews) {
		return nil
	}

	bv := doc.BufferViews[bvIdx]
	if bv.ByteOffset+bv.ByteLength > len(binBytes) {
		return nil
	}

	imgData := binBytes[bv.ByteOffset : bv.ByteOffset+bv.ByteLength]
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil
	}

	return fauxgl.NewImageTexture(img)
}

func loadMeshFile(path string) (*fauxgl.Mesh, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".glb" {
		return loadGLB(path)
	}
	return fauxgl.LoadMesh(path)
}

func rendermesh(filePath, texturePath string) ([]byte, error) {
	mesh, err := loadMeshFile(filePath)
	if err != nil {
		return nil, err
	}
	if mesh == nil || len(mesh.Triangles) == 0 {
		return nil, errors.New("mesh has no triangles")
	}

	mesh = mesh.Copy()
	mesh.BiUnitCube()
	mesh.SmoothNormalsThreshold(fauxgl.Radians(30))

	context := contextPool.Get().(*fauxgl.Context)
	defer contextPool.Put(context)
	context.ClearColorBufferWith(fauxgl.Transparent)
	context.ClearDepthBuffer()
	context.AlphaBlend = true

	eye := fauxgl.Vector{X: 2.2, Y: 1.8, Z: 2.4}
	look := fauxgl.Vector{X: 0.0, Y: 0.0, Z: 0.0}
	up := fauxgl.Vector{X: 0.0, Y: 1.0, Z: 0.0}
	aspect := float64(Width) / float64(Height)
	fov := 48.0

	view := fauxgl.LookAt(eye, look, up)
	projection := fauxgl.Perspective(fov, aspect, 0.1, 1000)
	matrixcm := projection.Mul(view)

	b := mesh.BoundingBox()
	corners := []fauxgl.Vector{
		{X: b.Min.X, Y: b.Min.Y, Z: b.Min.Z},
		{X: b.Min.X, Y: b.Min.Y, Z: b.Max.Z},
		{X: b.Min.X, Y: b.Max.Y, Z: b.Min.Z},
		{X: b.Min.X, Y: b.Max.Y, Z: b.Max.Z},
		{X: b.Max.X, Y: b.Min.Y, Z: b.Min.Z},
		{X: b.Max.X, Y: b.Min.Y, Z: b.Max.Z},
		{X: b.Max.X, Y: b.Max.Y, Z: b.Min.Z},
		{X: b.Max.X, Y: b.Max.Y, Z: b.Max.Z},
	}
	maxNDC := 0.0
	for _, p := range corners {
		nx, ny, _, w := getNDC(matrixcm, p)
		if w > 0.01 {
			if math.Abs(nx) > maxNDC {
				maxNDC = math.Abs(nx)
			}
			if math.Abs(ny) > maxNDC {
				maxNDC = math.Abs(ny)
			}
		}
	}
	if maxNDC > 0.85 {
		factor := maxNDC / 0.80
		newTan := math.Tan(fov*math.Pi/360.0) * factor
		fov = 2 * math.Atan(newTan) * 180.0 / math.Pi
		projection = fauxgl.Perspective(fov, aspect, 0.1, 1000)
		matrixcm = projection.Mul(view)
	}

	var tex fauxgl.Texture
	if texturePath != "" {
		tex, _ = fauxgl.LoadTexture(texturePath)
	}
	if tex == nil && strings.ToLower(filepath.Ext(filePath)) == ".glb" {
		tex = extractGLBTexture(filePath)
	}

	if tex != nil {
		meshColor := parsec("#FFFFFF")
		context.Shader = createGouraudShader(matrixcm, eye, meshColor, tex)
	} else {
		meshColor := parsec("#B8B8B8")
		context.Shader = createGouraudShader(matrixcm, eye, meshColor, nil)
	}

	context.DrawMesh(mesh)

	img := downsample(context.ColorBuffer, ScaleFactor)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func RenderMesh(filePath string) ([]byte, error) {
	return RenderMeshWithTexture(filePath, "")
}

func RenderMeshWithTexture(filePath, texturePath string) ([]byte, error) {
	req := RenderRequest{
		MeshPath:    filePath,
		TexturePath: texturePath,
	}
	return submitRenderJob(req, "mesh render timeout")
}

func RenderMeshAsset(db *sql.DB, assetID int) ([]byte, error) {
	if db == nil {
		return nil, errors.New("database is not connected")
	}

	var filePath string
	var assetType string
	err := db.QueryRow("SELECT file_path, type FROM assets WHERE id = ?", assetID).Scan(&filePath, &assetType)
	if err != nil {
		return nil, err
	}
	if assetType != "mesh" {
		return nil, errors.New("asset is not a mesh")
	}

	fullPath := filepath.Join("./uploads/assets", filepath.Clean(filePath))
	return RenderMesh(fullPath)
}