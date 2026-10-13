package renderer

import (
	"bytes"
	"image/png"
	"log"
	"math"

	"github.com/fogleman/fauxgl"
)

const toolArmFile = LArmFile + "Tool"

type BodyPart struct {
	Mesh    *fauxgl.Mesh
	Corners []fauxgl.Vector
}

type LoadedHat struct {
	Mesh *fauxgl.Mesh
	Tex  fauxgl.Texture
	ID   int
}

type hatSet struct {
	hats    []LoadedHat
	preview *fauxgl.Mesh
	box     *fauxgl.Box
}

type bodyTextures struct {
	face   fauxgl.Texture
	shirt  fauxgl.Texture
	pants  fauxgl.Texture
	tshirt fauxgl.Texture
}

var (
	bodyParts = make(map[string]*BodyPart)
	mHead     fauxgl.Matrix
	mTool     fauxgl.Matrix
)

func initBodyParts() {
	headCenter := fauxgl.Vector{}
	if mesh, ok := preloadmeshes[HeadFile]; ok {
		headCenter = mesh.BoundingBox().Center()
	}
	mHead = fauxgl.Identity().
		Translate(headCenter.Negate()).
		Scale(fauxgl.Vector{X: 1, Y: 1, Z: 1}).
		Translate(fauxgl.Vector{X: 0, Y: 3, Z: 0})

	mTool = fauxgl.Translate(fauxgl.Vector{X: 0, Y: -1.5, Z: 0})

	if mesh, ok := preloadmeshes[LArmFile]; ok {
		c := mesh.BoundingBox().Center()
		toolArm := fauxgl.Identity().
			Translate(c.Negate()).
			Rotate(fauxgl.Vector{X: 1, Y: 0, Z: 0}, math.Pi/2).
			Translate(c).
			Translate(fauxgl.Vector{X: 0, Y: -1, Z: 0.4})
		if part := makePart(LArmFile, toolArm); part != nil {
			bodyParts[toolArmFile] = part
		}
	}

	partMatrices := map[string]fauxgl.Matrix{
		HeadFile:   mHead,
		TorsoFile:  fauxgl.Translate(fauxgl.Vector{X: 0, Y: -1.5, Z: 0}),
		LArmFile:   fauxgl.Translate(fauxgl.Vector{X: 0, Y: -1.5, Z: 0}),
		RArmFile:   fauxgl.Translate(fauxgl.Vector{X: 0, Y: -1.5, Z: 0}),
		LLegFile:   fauxgl.Translate(fauxgl.Vector{X: 0, Y: -1.5, Z: 0}),
		RLegFile:   fauxgl.Translate(fauxgl.Vector{X: 0, Y: -1.5, Z: 0}),
		TShirtFile: fauxgl.Translate(fauxgl.Vector{X: 0, Y: -1.5, Z: 0.02}),
	}
	for name, mat := range partMatrices {
		if part := makePart(name, mat); part != nil {
			bodyParts[name] = part
		}
	}
}

func makePart(name string, mat fauxgl.Matrix) *BodyPart {
	mesh, ok := preloadmeshes[name]
	if !ok {
		return nil
	}
	return &BodyPart{
		Mesh:    transformMesh(mesh, mat),
		Corners: meshCorners(mesh, mat),
	}
}

func transformMesh(m *fauxgl.Mesh, mat fauxgl.Matrix) *fauxgl.Mesh {
	mesh := m.Copy()
	mesh.Transform(mat)
	return mesh
}

func meshCorners(m *fauxgl.Mesh, mat fauxgl.Matrix) []fauxgl.Vector {
	b := m.BoundingBox()
	return []fauxgl.Vector{
		mat.MulPosition(fauxgl.Vector{X: b.Min.X, Y: b.Min.Y, Z: b.Min.Z}),
		mat.MulPosition(fauxgl.Vector{X: b.Min.X, Y: b.Min.Y, Z: b.Max.Z}),
		mat.MulPosition(fauxgl.Vector{X: b.Min.X, Y: b.Max.Y, Z: b.Min.Z}),
		mat.MulPosition(fauxgl.Vector{X: b.Min.X, Y: b.Max.Y, Z: b.Max.Z}),
		mat.MulPosition(fauxgl.Vector{X: b.Max.X, Y: b.Min.Y, Z: b.Min.Z}),
		mat.MulPosition(fauxgl.Vector{X: b.Max.X, Y: b.Min.Y, Z: b.Max.Z}),
		mat.MulPosition(fauxgl.Vector{X: b.Max.X, Y: b.Max.Y, Z: b.Min.Z}),
		mat.MulPosition(fauxgl.Vector{X: b.Max.X, Y: b.Max.Y, Z: b.Max.Z}),
	}
}

func renderav(req RenderRequest) ([]byte, error) {
	context := contextPool.Get().(*fauxgl.Context)
	defer contextPool.Put(context)
	context.ClearColorBufferWith(fauxgl.Transparent)
	context.ClearDepthBuffer()
	context.AlphaBlend = true

	hats := loadHats(req)
	toolMesh, toolTex := loadTool(req)

	eye, look := cameraFor(req, hats.box)
	fov := fitFov(req, hats, toolMesh, eye, look)
	view := fauxgl.LookAt(eye, look, fauxgl.Vector{X: 0, Y: 1, Z: 0})
	projection := fauxgl.Perspective(fov, float64(Width)/float64(Height), 0.1, 1000)
	matrixcm := projection.Mul(view)

	textures := loadBodyTextures(req)
	drawAvatar(context, matrixcm, eye, req, hats, toolMesh, toolTex, textures)

	img := downsample(context.ColorBuffer, ScaleFactor)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func loadHats(req RenderRequest) hatSet {
	var hs hatSet

	if req.PreviewType == "hat" {
		if req.PreviewObj != "" {
			if m, err := fauxgl.LoadOBJ(req.PreviewObj); err == nil {
				hs.preview = m
			}
		} else if req.Hat1ID > 0 {
			if m, err := fetchMesh("hat", req.Hat1ID); err == nil {
				hs.preview = m
			}
		}
		if hs.preview != nil {
			b := mHead.MulBox(hs.preview.BoundingBox())
			hs.box = &b
		}
	}

	for _, hid := range []int{req.Hat1ID, req.Hat2ID, req.Hat3ID, req.Hat4ID, req.Hat5ID} {
		if hid <= 0 {
			continue
		}
		var mesh *fauxgl.Mesh
		if req.PreviewType == "hat" && hid == req.Hat1ID && hs.preview != nil {
			mesh = hs.preview
		} else {
			m, err := fetchMesh("hat", hid)
			if err != nil && m == nil {
				log.Printf("failed loading hat mesh %d: %v", hid, err)
				continue
			}
			mesh = m
		}
		tex, err := fetchTexture("hat", hid, "")
		if err != nil {
			log.Printf("failed loading hat texture %d: %v", hid, err)
		}
		hs.hats = append(hs.hats, LoadedHat{Mesh: mesh, Tex: tex, ID: hid})
	}
	return hs
}

func loadTool(req RenderRequest) (*fauxgl.Mesh, fauxgl.Texture) {
	if req.PreviewType == "gear" && req.PreviewObj != "" {
		m, err := fauxgl.LoadOBJ(req.PreviewObj)
		if err != nil {
			return nil, nil
		}
		if req.PreviewTexture != "" {
			tex, err := fauxgl.LoadTexture(req.PreviewTexture)
			if err != nil {
				return nil, nil
			}
			return m, tex
		}
		return m, nil
	}
	if req.ToolID > 0 {
		m, err := fetchMesh("gear", req.ToolID)
		if err != nil {
			return nil, nil
		}
		tex, _ := fetchTexture("gear", req.ToolID, "")
		return m, tex
	}
	return nil, nil
}

func cameraFor(req RenderRequest, hatBox *fauxgl.Box) (fauxgl.Vector, fauxgl.Vector) {
	eye := fauxgl.Vector{X: CamPosX, Y: CamPosY, Z: CamPosZ}
	look := fauxgl.Vector{X: CamLookX, Y: CamLookY, Z: CamLookZ}
	if req.IsTool {
		eye = fauxgl.Vector{X: 2, Y: 4.2, Z: 4.8}
	}
	if req.PreviewType == "hat" && hatBox != nil {
		c := hatBox.Center()
		s := hatBox.Size()
		maxS := math.Max(s.X, math.Max(s.Y, s.Z))
		look = fauxgl.Vector{X: c.X * 0.5, Y: c.Y - 1.0, Z: c.Z * 0.5}
		eye = fauxgl.Vector{X: c.X + 1.2 + maxS*0.5, Y: c.Y + 0.5 + maxS*0.2, Z: c.Z + 2.5 + maxS*1.2}
	} else if req.PreviewType == "face" || req.PreviewType == "faces" {
		look = fauxgl.Vector{X: 0.0, Y: 3.0, Z: 0.0}
		eye = fauxgl.Vector{X: 1.0, Y: 3.2, Z: 2.5}
	}
	return eye, look
}

func fitFov(req RenderRequest, hs hatSet, toolMesh *fauxgl.Mesh, eye, look fauxgl.Vector) float64 {
	defaultFov := 2 * math.Atan(math.Tan(BaseFOV*math.Pi/360.0)/DefaultZoom) * 180.0 / math.Pi
	if req.PreviewType == "face" || req.PreviewType == "faces" || req.PreviewType == "hat" {
		return defaultFov
	}

	view := fauxgl.LookAt(eye, look, fauxgl.Vector{X: 0, Y: 1, Z: 0})
	matrixcm := fauxgl.Perspective(defaultFov, float64(Width)/float64(Height), 0.1, 1000).Mul(view)

	corners := make([]fauxgl.Vector, 0, 56+8*len(hs.hats)+8)
	for _, name := range []string{HeadFile, TorsoFile, RArmFile, LLegFile, RLegFile, TShirtFile} {
		if part, ok := bodyParts[name]; ok && part != nil {
			corners = append(corners, part.Corners...)
		}
	}
	armName := LArmFile
	if req.IsTool {
		armName = toolArmFile
	}
	if part, ok := bodyParts[armName]; ok && part != nil {
		corners = append(corners, part.Corners...)
	}
	for _, h := range hs.hats {
		corners = append(corners, meshCorners(h.Mesh, mHead)...)
	}
	if toolMesh != nil {
		corners = append(corners, meshCorners(toolMesh, mTool)...)
	}

	maxNDC := 0.0
	for _, p := range corners {
		nx, ny, _, w := getNDC(matrixcm, p)
		if w <= 0.01 {
			continue
		}
		if math.Abs(nx) > maxNDC {
			maxNDC = math.Abs(nx)
		}
		if math.Abs(ny) > maxNDC {
			maxNDC = math.Abs(ny)
		}
	}
	if maxNDC > 0.99 {
		factor := maxNDC * AutoZoomMargin
		newTan := (math.Tan(BaseFOV*math.Pi/360.0) / DefaultZoom) * factor
		return 2 * math.Atan(newTan) * 180.0 / math.Pi
	}
	return defaultFov
}

func loadBodyTextures(req RenderRequest) bodyTextures {
	var textures bodyTextures
	if (req.PreviewType == "face" || req.PreviewType == "faces") && req.PreviewTexture != "" {
		textures.face, _ = fauxgl.LoadTexture(req.PreviewTexture)
	} else {
		textures.face, _ = fetchTexture("faces", req.FaceID, FacePath+"0.png")
	}
	if (req.PreviewType == "shirt" || req.PreviewType == "shirts") && req.PreviewTexture != "" {
		textures.shirt, _ = fauxgl.LoadTexture(req.PreviewTexture)
	} else if req.ShirtID > 0 {
		var err error
		textures.shirt, err = fetchTexture("shirts", req.ShirtID, "")
		if err != nil {
			log.Printf("failed loading shirt: %v", err)
		}
	}
	if (req.PreviewType == "pant" || req.PreviewType == "pants") && req.PreviewTexture != "" {
		textures.pants, _ = fauxgl.LoadTexture(req.PreviewTexture)
	} else if req.PantsID > 0 {
		var err error
		textures.pants, err = fetchTexture("pants", req.PantsID, "")
		if err != nil {
			log.Printf("failed loading pants: %v", err)
		}
	}
	if (req.PreviewType == "tshirt" || req.PreviewType == "tshirts") && req.PreviewTexture != "" {
		textures.tshirt, _ = fauxgl.LoadTexture(req.PreviewTexture)
	} else if req.TShirtID > 0 {
		var err error
		textures.tshirt, err = fetchTexture("tshirts", req.TShirtID, "")
		if err != nil {
			log.Printf("failed loading tshirt: %v", err)
		}
	}
	return textures
}

func drawAvatar(context *fauxgl.Context, matrixcm fauxgl.Matrix, eye fauxgl.Vector, req RenderRequest, hs hatSet, toolMesh *fauxgl.Mesh, toolTex fauxgl.Texture, textures bodyTextures) {
	drawPart := func(part *BodyPart, colorHex string, texture fauxgl.Texture) {
		if part == nil {
			return
		}
		context.Shader = createGouraudShader(matrixcm, eye, parsec(colorHex), texture)
		context.DrawMesh(part.Mesh)
	}

	var headTex fauxgl.Texture
	if textures.face != nil {
		headTex = CompositeTexture{Layers: []fauxgl.Texture{textures.face}, Color: parsec(req.HeadColor)}
	}
	drawPart(bodyParts[HeadFile], req.HeadColor, headTex)

	for _, h := range hs.hats {
		hatMesh := h.Mesh.Copy()
		context.Shader = createGouraudShader(matrixcm, eye, parsec("#FFFFFF"), h.Tex)
		hatMesh.Transform(mHead)
		context.DrawMesh(hatMesh)
	}

	if req.PreviewType == "hat" && req.PreviewObj != "" && hs.preview != nil {
		var hatTex fauxgl.Texture
		if req.PreviewTexture != "" {
			hatTex, _ = fauxgl.LoadTexture(req.PreviewTexture)
		}
		previewMesh := hs.preview.Copy()
		context.Shader = createGouraudShader(matrixcm, eye, parsec("#FFFFFF"), hatTex)
		previewMesh.Transform(mHead)
		context.DrawMesh(previewMesh)
	}

	var torsoTex fauxgl.Texture
	if textures.pants != nil || textures.shirt != nil {
		layers := make([]fauxgl.Texture, 0, 2)
		if textures.pants != nil {
			layers = append(layers, textures.pants)
		}
		if textures.shirt != nil {
			layers = append(layers, textures.shirt)
		}
		torsoTex = CompositeTexture{Layers: layers, Color: parsec(req.TorsoColor)}
	}
	drawPart(bodyParts[TorsoFile], req.TorsoColor, torsoTex)

	var rArmTex fauxgl.Texture
	if textures.shirt != nil {
		rArmTex = CompositeTexture{Layers: []fauxgl.Texture{textures.shirt}, Color: parsec(req.RightArmColor)}
	}
	drawPart(bodyParts[RArmFile], req.RightArmColor, rArmTex)

	armName := LArmFile
	if req.IsTool {
		armName = toolArmFile
	}
	var lArmTex fauxgl.Texture
	if textures.shirt != nil {
		lArmTex = CompositeTexture{Layers: []fauxgl.Texture{textures.shirt}, Color: parsec(req.LeftArmColor)}
	}
	drawPart(bodyParts[armName], req.LeftArmColor, lArmTex)

	var lLegTex fauxgl.Texture
	if textures.pants != nil {
		lLegTex = CompositeTexture{Layers: []fauxgl.Texture{textures.pants}, Color: parsec(req.LeftLegColor)}
	}
	drawPart(bodyParts[LLegFile], req.LeftLegColor, lLegTex)

	var rLegTex fauxgl.Texture
	if textures.pants != nil {
		rLegTex = CompositeTexture{Layers: []fauxgl.Texture{textures.pants}, Color: parsec(req.RightLegColor)}
	}
	drawPart(bodyParts[RLegFile], req.RightLegColor, rLegTex)

	if toolMesh != nil {
		toolCopy := toolMesh.Copy()
		context.Shader = createGouraudShader(matrixcm, eye, parsec("#FFFFFF"), toolTex)
		toolCopy.Transform(mTool)
		context.DrawMesh(toolCopy)
	}

	if textures.tshirt != nil {
		drawPart(bodyParts[TShirtFile], "#FFFFFF", textures.tshirt)
	}
}