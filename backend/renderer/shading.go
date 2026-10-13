package renderer

import (
	"encoding/hex"
	"math"

	"github.com/fogleman/fauxgl"
)

var lightDirection = fauxgl.Vector{X: -1, Y: 1, Z: 1}

func createGouraudShader(matrixcm fauxgl.Matrix, eye fauxgl.Vector, objColor fauxgl.Color, texture fauxgl.Texture) *fauxgl.PhongShader {
	shader := fauxgl.NewPhongShader(matrixcm, lightDirection, eye)
	shader.AmbientColor = fauxgl.Color{R: Brightness, G: Brightness, B: Brightness, A: 1}
	shader.DiffuseColor = fauxgl.Color{R: 0.35, G: 0.35, B: 0.35, A: 1}
	specularIntensity := math.Max(0, 1.0-Roughness)
	shader.SpecularColor = fauxgl.Color{R: specularIntensity, G: specularIntensity, B: specularIntensity, A: 1}
	shader.ObjectColor = objColor
	shader.Texture = texture
	return shader
}

func parsec(hexst string) fauxgl.Color {
	if len(hexst) > 0 && hexst[0] == '#' {
		hexst = hexst[1:]
	}
	c := fauxgl.Color{R: 1, G: 1, B: 1, A: 1}
	if len(hexst) == 6 {
		bytes, err := hex.DecodeString(hexst)
		if err == nil {
			c.R = float64(bytes[0]) / 255.0
			c.G = float64(bytes[1]) / 255.0
			c.B = float64(bytes[2]) / 255.0
		}
	}
	return c
}

func getNDC(m fauxgl.Matrix, v fauxgl.Vector) (float64, float64, float64, float64) {
	x := m.X00*v.X + m.X01*v.Y + m.X02*v.Z + m.X03
	y := m.X10*v.X + m.X11*v.Y + m.X12*v.Z + m.X13
	z := m.X20*v.X + m.X21*v.Y + m.X22*v.Z + m.X23
	w := m.X30*v.X + m.X31*v.Y + m.X32*v.Z + m.X33
	if w != 0 {
		return x / w, y / w, z / w, w
	}
	return x, y, z, w
}