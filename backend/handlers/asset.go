package handlers

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/config"
	"buildoverse/backend/models"
	"buildoverse/backend/renderer"
	"buildoverse/backend/service"
)

const (
	assetUploadDir     = "./uploads/assets"
	fallbackRejectPath = "./static/useful/reject.png"
	fallbackTimePath   = "./static/useful/time.png"
)

var assetTypeExtensions = map[string]map[string]bool{
	models.AssetTypeImage: {".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true},
	models.AssetTypeMesh:  {".glb": true, ".obj": true},
	models.AssetTypeSound: {".mp3": true, ".wav": true, ".ogg": true},
}

var assetSizeLimits = map[string]int64{
	models.AssetTypeImage: 8 << 20,
	models.AssetTypeMesh:  25 << 20,
	models.AssetTypeSound: 15 << 20,
}

func assetContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".glb":
		return "model/gltf-binary"
	case ".obj":
		return "text/plain"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	default:
		return "application/octet-stream"
	}
}

func parseAssetID(rawID string) (int, error) {
	clean := strings.TrimSpace(rawID)
	if dotIdx := strings.Index(clean, "."); dotIdx != -1 {
		clean = clean[:dotIdx]
	}
	return strconv.Atoi(clean)
}

func validateAssetContent(assetType, ext string, header []byte) error {
	switch assetType {
	case models.AssetTypeImage:
		valid := bytes.HasPrefix(header, []byte{0xFF, 0xD8}) ||
			bytes.HasPrefix(header, []byte{0x89, 'P', 'N', 'G'}) ||
			bytes.HasPrefix(header, []byte("GIF8")) ||
			len(header) > 11 && bytes.HasPrefix(header, []byte("RIFF")) && string(header[8:12]) == "WEBP"
		if !valid {
			return errors.New("invalid image file")
		}
		if ext == ".png" && !bytes.HasPrefix(header, []byte{0x89, 'P', 'N', 'G'}) {
			return errors.New("invalid PNG file")
		}
		if ext == ".gif" && !bytes.HasPrefix(header, []byte("GIF8")) {
			return errors.New("invalid GIF file")
		}
	case models.AssetTypeMesh:
		if ext == ".glb" && !bytes.HasPrefix(header, []byte("glTF")) {
			return errors.New("invalid GLB file")
		}
	case models.AssetTypeSound:
		valid := bytes.HasPrefix(header, []byte("ID3")) ||
			(len(header) > 1 && header[0] == 0xFF && header[1]&0xE0 == 0xE0) ||
			len(header) > 11 && bytes.HasPrefix(header, []byte("RIFF")) && string(header[8:12]) == "WAVE" ||
			bytes.HasPrefix(header, []byte("OggS"))
		if !valid {
			return errors.New("invalid audio file")
		}
	}
	return nil
}

func isAuthorizedForAsset(c fiber.Ctx, asset *models.Asset) bool {
	if config.Global != nil {
		expectedServerKey := strings.Trim(strings.TrimSpace(config.Global.GameserverAPIKey), "\"")
		if expectedServerKey != "" {
			providedServerKey := c.Get("X-Gameserver-Key")
			if providedServerKey == "" {
				providedServerKey = c.Get("Gameserver-Key")
			}
			if providedServerKey == "" {
				providedServerKey = c.Get("X-API-Key")
			}
			if providedServerKey == "" {
				providedServerKey = c.Query("gameserver_key")
			}
			if providedServerKey == "" {
				providedServerKey = c.Query("api_key")
			}
			if providedServerKey == "" {
				providedServerKey = c.Query("key")
			}
			if providedServerKey == "" {
				authHeader := c.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					providedServerKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}
			providedServerKey = strings.Trim(strings.TrimSpace(providedServerKey), "\"")
			if providedServerKey != "" && subtle.ConstantTimeCompare([]byte(providedServerKey), []byte(expectedServerKey)) == 1 {
				return true
			}
		}
	}

	providedUkey := c.Query("ukey")
	if providedUkey == "" {
		providedUkey = c.Query("unikey")
	}
	if providedUkey == "" {
		providedUkey = c.Query("user_key")
	}
	if providedUkey == "" {
		providedUkey = c.Get("X-User-Key")
	}
	if providedUkey == "" {
		providedUkey = c.Get("X-Unikey")
	}
	if providedUkey == "" {
		providedUkey = c.Get("X-Ukey")
	}
	providedUkey = strings.Trim(strings.TrimSpace(providedUkey), "\"")

	if providedUkey != "" && service.User != nil {
		user, err := service.User.GetUserByUnikey(providedUkey)
		if err == nil && user != nil {
			if user.ID == asset.UID || user.HasAdminAccess() {
				return true
			}
		}
	}

	username := GetActiveUser(c)
	if username != "" && service.User != nil {
		user, err := service.User.GetUserByUsername(username)
		if err == nil && user != nil {
			if user.ID == asset.UID || user.HasAdminAccess() {
				return true
			}
		}
	}

	return false
}

func AssetsPage(c fiber.Ctx) error {
	username := GetActiveUser(c)
	if username == "" {
		return c.Redirect().To("/login")
	}

	user, err := service.User.GetUserByUsername(username)
	if err != nil || user == nil {
		return c.Redirect().To("/login")
	}

	var assets []*models.Asset
	if service.Asset != nil {
		assets, _ = service.Asset.GetUserAssets(user.ID)
	}
	if assets == nil {
		assets = []*models.Asset{}
	}

	return Render(c, "pages/assets", fiber.Map{
		"Title":    "My Assets - Buildoverse",
		"Username": username,
		"Assets":   assets,
	}, "layouts/main")
}

func AssetUploadPost(c fiber.Ctx) error {
	username := GetActiveUser(c)
	if username == "" {
		return c.Redirect().To("/login")
	}

	user, err := service.User.GetUserByUsername(username)
	if err != nil || user == nil {
		return c.Redirect().To("/login")
	}

	assetType, _ := models.NormalizeAssetType(c.FormValue("type"))

	redirectError := func(msg string) error {
		return c.Redirect().To("/create?type=asset&asset=" + assetType + "&error=" + url.QueryEscape(msg))
	}

	accountAge := time.Since(user.CreationDate)
	if assetType == models.AssetTypeImage {
		if accountAge < 3*24*time.Hour && !user.HasAdminAccess() {
			return redirectError("Your account must be at least 3 days old to upload image assets")
		}
	} else if assetType == models.AssetTypeMesh || assetType == models.AssetTypeSound {
		if accountAge < 7*24*time.Hour && !user.HasAdminAccess() {
			return redirectError("Your account must be at least 7 days old to upload " + models.AssetTypeNames[assetType] + " assets")
		}
	}

	extensions, ok := assetTypeExtensions[assetType]
	if !ok {
		return redirectError("Invalid asset type")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return redirectError("Missing file")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !extensions[ext] {
		return redirectError("Invalid file type for " + models.AssetTypeNames[assetType])
	}

	if fileHeader.Size <= 0 {
		return redirectError("Empty file")
	}
	if fileHeader.Size > assetSizeLimits[assetType] {
		return redirectError("File is too large")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return redirectError("Could not read upload")
	}
	defer src.Close()

	header := make([]byte, 512)
	n, _ := io.ReadFull(src, header)
	header = header[:n]
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return redirectError("Could not read upload")
	}

	if err := validateAssetContent(assetType, ext, header); err != nil {
		return redirectError(err.Error())
	}

	fileName, err := models.GenerateRandomString(24)
	if err != nil {
		return redirectError("Could not save upload")
	}

	dir := filepath.Join(assetUploadDir, strconv.Itoa(user.ID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return redirectError("Could not save upload")
	}

	relPath := filepath.Join(strconv.Itoa(user.ID), fileName+ext)
	fullSavedPath := filepath.Join(assetUploadDir, relPath)
	dst, err := os.Create(fullSavedPath)
	if err != nil {
		return redirectError("Could not save upload")
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(fullSavedPath)
		return redirectError("Could not save upload")
	}
	dst.Close()

	var relTexPath string
	if assetType == models.AssetTypeMesh {
		if texHeader, err := c.FormFile("texture"); err == nil && texHeader != nil && texHeader.Size > 0 {
			texExt := strings.ToLower(filepath.Ext(texHeader.Filename))
			if validExtensions := assetTypeExtensions[models.AssetTypeImage]; validExtensions[texExt] {
				if texHeader.Size <= assetSizeLimits[models.AssetTypeImage] {
					if texSrc, err := texHeader.Open(); err == nil {
						texHdr := make([]byte, 512)
						tn, _ := io.ReadFull(texSrc, texHdr)
						texHdr = texHdr[:tn]
						_, _ = texSrc.Seek(0, io.SeekStart)
						if err := validateAssetContent(models.AssetTypeImage, texExt, texHdr); err == nil {
							relTexPath = filepath.Join(strconv.Itoa(user.ID), fileName+"_tex"+texExt)
							fullTexPath := filepath.Join(assetUploadDir, relTexPath)
							if dstTex, err := os.Create(fullTexPath); err == nil {
								_, _ = io.Copy(dstTex, texSrc)
								dstTex.Close()
							}
						}
						texSrc.Close()
					}
				}
			}
		}
	}

	if service.Asset == nil {
		os.Remove(fullSavedPath)
		if relTexPath != "" {
			os.Remove(filepath.Join(assetUploadDir, relTexPath))
		}
		return redirectError("Asset service unavailable")
	}

	assetID, err := service.Asset.CreateAsset(user.ID, c.FormValue("name"), c.FormValue("description"), assetType, relPath)
	if err != nil {
		os.Remove(fullSavedPath)
		if relTexPath != "" {
			os.Remove(filepath.Join(assetUploadDir, relTexPath))
		}
		return redirectError(err.Error())
	}

	if assetType == models.AssetTypeMesh {
		var fullTexPath string
		if relTexPath != "" {
			fullTexPath = filepath.Join(assetUploadDir, relTexPath)
		}
		if imgBytes, err := renderer.RenderMeshWithTexture(fullSavedPath, fullTexPath); err == nil && len(imgBytes) > 0 {
			cachePath := filepath.Join("static", "renders", "meshes", strconv.Itoa(assetID)+".png")
			_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
			_ = os.WriteFile(cachePath, imgBytes, 0644)
		}
	}

	return c.Redirect().To("/create/assets")
}

func AssetFileGet(c fiber.Ctx) error {
	assetID, err := parseAssetID(c.Params("id"))
	if err != nil || assetID <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid asset ID")
	}

	if service.Asset == nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Asset service unavailable")
	}

	asset, err := service.Asset.GetByID(assetID)
	if err != nil || asset == nil {
		return c.Status(fiber.StatusNotFound).SendString("Asset not found")
	}

	if !isAuthorizedForAsset(c, asset) {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}

	if asset.ApprovalState == models.AssetApprovalRejected {
		c.Set("Content-Type", "image/png")
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		return c.SendFile(fallbackRejectPath)
	}

	if asset.ApprovalState != models.AssetApprovalApproved {
		c.Set("Content-Type", "image/png")
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		return c.SendFile(fallbackTimePath)
	}

	path := filepath.Join(assetUploadDir, filepath.Clean(asset.FilePath))
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(assetUploadDir)+string(os.PathSeparator)) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid asset path")
	}

	c.Set("Content-Type", assetContentType(filepath.Ext(asset.FilePath)))
	c.Set("Cache-Control", "public, max-age=86400")
	return c.SendFile(path)
}

func AssetRenderGet(c fiber.Ctx) error {
	assetID, err := parseAssetID(c.Params("id"))
	if err != nil || assetID <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid asset ID")
	}

	if service.Asset == nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Asset service unavailable")
	}

	asset, err := service.Asset.GetByID(assetID)
	if err != nil || asset == nil {
		return c.Status(fiber.StatusNotFound).SendString("Asset not found")
	}

	if !isAuthorizedForAsset(c, asset) {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}

	if asset.ApprovalState == models.AssetApprovalRejected {
		c.Set("Content-Type", "image/png")
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		return c.SendFile(fallbackRejectPath)
	}

	if asset.ApprovalState != models.AssetApprovalApproved {
		c.Set("Content-Type", "image/png")
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		return c.SendFile(fallbackTimePath)
	}

	if asset.Type == models.AssetTypeImage {
		return AssetFileGet(c)
	}

	if asset.Type != models.AssetTypeMesh {
		return c.Status(fiber.StatusBadRequest).SendString("Asset is not a mesh")
	}

	cachePath := filepath.Join("static", "renders", "meshes", strconv.Itoa(assetID)+".png")
	if imgBytes, err := os.ReadFile(cachePath); err == nil {
		c.Set("Content-Type", "image/png")
		c.Set("Cache-Control", "public, max-age=86400")
		return c.Send(imgBytes)
	}

	fullPath := filepath.Join(assetUploadDir, filepath.Clean(asset.FilePath))
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(assetUploadDir)+string(os.PathSeparator)) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid asset path")
	}

	var fullTexPath string
	baseWithoutExt := strings.TrimSuffix(fullPath, filepath.Ext(fullPath))
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp"} {
		possibleTex := baseWithoutExt + "_tex" + ext
		if _, err := os.Stat(possibleTex); err == nil {
			fullTexPath = possibleTex
			break
		}
	}

	imgBytes, err := renderer.RenderMeshWithTexture(fullPath, fullTexPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
	_ = os.WriteFile(cachePath, imgBytes, 0644)

	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(imgBytes)
}

func AdminAssetsAPI(c fiber.Ctx) error {
	_, err := getAdminUser(c)
	if err != nil {
		return adminAPIError(c, err)
	}

	category := c.Query("category")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))

	if service.Asset == nil {
		return c.JSON(fiber.Map{
			"assets":      []*models.Asset{},
			"total":       0,
			"page":        page,
			"limit":       limit,
			"total_pages": 1,
		})
	}

	assets, total, err := service.Asset.GetQueue(category, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	totalPages := 1
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return c.JSON(fiber.Map{
		"assets":      assets,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func AdminAssetReviewPost(c fiber.Ctx) error {
	adminUser, err := getAdminUser(c)
	if err != nil {
		return adminAuthError(c, err)
	}

	if !csrfValid(c) {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	assetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || assetID <= 0 {
		return adminModerationError(c, "Invalid asset ID")
	}

	if service.Asset == nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Asset service unavailable")
	}

	asset, err := service.Asset.GetByID(assetID)
	if err != nil || asset == nil {
		return adminModerationError(c, "Asset not found")
	}

	action := c.Params("action")
	state := models.AssetApprovalPending
	actionType := models.ActionApproveAsset
	switch action {
	case "approve":
		state = models.AssetApprovalApproved
		actionType = models.ActionApproveAsset
	case "reject":
		state = models.AssetApprovalRejected
		actionType = models.ActionRejectAsset
	default:
		return adminModerationError(c, "Unknown review action")
	}

	note := strings.TrimSpace(c.FormValue("note"))
	if err := service.Asset.Review(assetID, state, adminUser.ID, note); err != nil {
		return adminModerationError(c, err.Error())
	}

	if service.ModHistory != nil {
		_ = service.ModHistory.RecordAssetReview(adminUser, asset.UID, asset.ID, asset.Name, actionType, note)
	}

	if c.Get("Accept") == "application/json" || c.Get("X-Requested-With") == "XMLHttpRequest" || strings.HasPrefix(c.Path(), "/api/") {
		return c.JSON(fiber.Map{
			"success": true,
			"id":      assetID,
			"state":   state,
			"action":  action,
		})
	}

	return adminRedirectAssets(c)
}

func adminRedirectAssets(c fiber.Ctx) error {
	path := "/admin?tab=assets"
	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", path)
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect().To(path)
}