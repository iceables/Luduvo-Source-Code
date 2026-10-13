package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"buildoverse/backend/config"
	"buildoverse/backend/models"
	"buildoverse/backend/repository"
)

type AuthService struct {
	userRepo    *repository.UserRepository
	replayCache *models.ReplayCache
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		replayCache: models.NewReplayCache(),
	}
}

func (s *AuthService) VerifyAltcha(payloadStr string) error {
	if payloadStr == "" {
		return errors.New("CAPTCHA is required")
	}

	decoded, err := base64.StdEncoding.DecodeString(payloadStr)
	if err != nil {
		return errors.New("invalid CAPTCHA encoding")
	}

	var payload struct {
		Algorithm string `json:"algorithm"`
		Challenge string `json:"challenge"`
		Number    int    `json:"number"`
		Salt      string `json:"salt"`
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return errors.New("invalid CAPTCHA format")
	}

	parts := strings.Split(payload.Salt, "?expires=")
	if len(parts) != 2 {
		return errors.New("invalid CAPTCHA salt format")
	}

	expiresUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return errors.New("invalid CAPTCHA expiration format")
	}

	if time.Now().Unix() > expiresUnix {
		return errors.New("CAPTCHA challenge has expired")
	}

	secret := config.Global.AltchaHMACKey

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload.Challenge))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(payload.Signature), []byte(expectedSignature)) {
		return errors.New("CAPTCHA signature verification failed")
	}

	h := sha256.New()
	h.Write([]byte(payload.Salt + strconv.Itoa(payload.Number)))
	expectchlng := hex.EncodeToString(h.Sum(nil))
	if payload.Challenge != expectchlng {
		return errors.New("CAPTCHA challenge verification failed")
	}

	if !s.replayCache.Add(payload.Signature, time.Unix(expiresUnix, 0)) {
		return errors.New("CAPTCHA challenge has already been used")
	}

	return nil
}

func (s *AuthService) GenerateAltchaChallenge() (map[string]any, error) {
	secret := config.Global.AltchaHMACKey
	expat := time.Now().Add(15 * time.Minute).Unix()

	saltBytes := make([]byte, 12)
	if _, err := rand.Read(saltBytes); err != nil {
		return nil, errors.New("failed to generate salt")
	}
	salt := hex.EncodeToString(saltBytes) + "?expires=" + strconv.FormatInt(expat, 10)

	maxNumber := 50000
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return nil, errors.New("failed to generate number")
	}
	number := (int(binary.BigEndian.Uint64(b)) % maxNumber) + 1

	h := sha256.New()
	h.Write([]byte(salt + strconv.Itoa(number)))
	challenge := hex.EncodeToString(h.Sum(nil))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(challenge))
	signature := hex.EncodeToString(mac.Sum(nil))

	return map[string]any{
		"algorithm": "SHA-256",
		"challenge": challenge,
		"salt":      salt,
		"signature": signature,
		"maxnumber": maxNumber,
	}, nil
}

func (s *AuthService) AuthenticateUser(identifier, password string) (*models.User, error) {
	var user *models.User
	var err error

	if strings.Contains(identifier, "@") {
		user, err = s.userRepo.GetByEmail(identifier)
	} else {
		user, err = s.userRepo.GetByUsername(identifier)
	}

	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("invalid username/email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid username/email or password")
	}

	_ = s.userRepo.UpdateLastOnline(user.ID)

	return user, nil
}

func (s *AuthService) RegisterUser(username, displayname, email, password, passwordConfirm string) (*models.User, error) {
	if username == "" || email == "" || password == "" {
		return nil, errors.New("All required fields must be filled!")
	}

	if password != passwordConfirm {
		return nil, errors.New("Passwords do not match")
	}

	if len(password) < 8 {
		return nil, errors.New("Password must be at least 8 characters long")
	}

	existingUser, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("username is already taken")
	}

	isOld, err := s.userRepo.IsOldUsername(username)
	if err != nil {
		return nil, err
	}
	if isOld {
		return nil, errors.New("username is already taken")
	}

	existingEmail, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, err
	}
	if existingEmail != nil {
		return nil, errors.New("email is already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	unikey, err := models.GenerateRandomString(25)
	if err != nil {
		return nil, err
	}

	if displayname == "" {
		displayname = username
	}

	user := &models.User{
		Username:     username,
		DisplayName:  displayname,
		Mail:         email,
		Password:     string(hashedPassword),
		Description:  "I am new to Buildoverse!",
		Unikey:       unikey,
		Power:        0,
		PrimaryClan:  0,
		NameColor:    "7423CB",
		Vermail:      "false",
		Vermc:        "false",
		Casom:        "false",
		Bits:         250,
		Bucks:        100,
		LastOnline:   time.Now(),
		CreationDate: time.Now(),
		Views:        0,
		FeedCaptchas: 0,
	}

	if err := s.userRepo.CreateUser(user, models.DefaultTorsoColor, models.DefaultLegColor); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) CreateRememberToken(userID int, duration time.Duration) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	rawToken := hex.EncodeToString(tokenBytes)

	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])
	expiresAt := time.Now().Add(duration)

	if err := s.userRepo.CreateRememberToken(userID, tokenHash, expiresAt); err != nil {
		return "", err
	}

	payload := fmt.Sprintf("%d:%s", userID, rawToken)
	return base64.StdEncoding.EncodeToString([]byte(payload)), nil
}

func (s *AuthService) ValidateRememberToken(encodedCookie string) (*models.User, error) {
	if encodedCookie == "" {
		return nil, nil
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCookie)
	if err != nil {
		return nil, nil
	}

	parts := strings.SplitN(string(decodedBytes), ":", 2)
	if len(parts) != 2 {
		return nil, nil
	}

	userID, err := strconv.Atoi(parts[0])
	if err != nil || userID <= 0 {
		return nil, nil
	}

	rawToken := parts[1]
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	expiresAt, err := s.userRepo.GetRememberTokenExpiry(userID, tokenHash)
	if err != nil {
		return nil, nil
	}

	if time.Now().After(expiresAt) {
		_ = s.userRepo.DeleteRememberToken(tokenHash)
		return nil, nil
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return nil, nil
	}

	_ = s.userRepo.UpdateLastOnline(user.ID)
	return user, nil
}

func (s *AuthService) RevokeRememberToken(encodedCookie string) {
	if encodedCookie == "" {
		return
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCookie)
	if err != nil {
		return
	}

	parts := strings.SplitN(string(decodedBytes), ":", 2)
	if len(parts) != 2 {
		return
	}

	rawToken := parts[1]
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	_ = s.userRepo.DeleteRememberToken(tokenHash)
}

func (s *AuthService) ValidateUkey(apiKey, ukey string) (*models.User, error) {
	apiKey = strings.TrimSpace(apiKey)
	apiKey = strings.Trim(apiKey, "\"")

	expectedKey := strings.TrimSpace(config.Global.GameserverAPIKey)
	expectedKey = strings.Trim(expectedKey, "\"")

	if apiKey == "" || subtle.ConstantTimeCompare([]byte(apiKey), []byte(expectedKey)) != 1 {
		return nil, errors.New("unauthorized")
	}

	if ukey == "" {
		return nil, errors.New("missing ukey")
	}

	user, err := s.userRepo.GetByUnikey(ukey)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("not found")
	}

	return user, nil
}
