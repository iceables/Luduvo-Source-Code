package repository

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"buildoverse/backend/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

const userSelectFields = "id, username, displayname, mail, password, description, unikey, power, primary_clan, namecolor, custom_css, vermail, vermc, casom, bits, bucks, last_online, creation_date, views, feed_captchas, email_verify_token, email_verify_expiry, COALESCE(pronouns, ''), COALESCE(socials, ''), music_id"

type scannable interface {
	Scan(dest ...any) error
}

func scanUser(s scannable) (*models.User, error) {
	var u models.User
	err := s.Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.Mail, &u.Password,
		&u.Description, &u.Unikey, &u.Power, &u.PrimaryClan, &u.NameColor,
		&u.CustomCSS, &u.Vermail, &u.Vermc, &u.Casom, &u.Bits,
		&u.Bucks, &u.LastOnline, &u.CreationDate, &u.Views, &u.FeedCaptchas,
		&u.EmailVerifyToken, &u.EmailVerifyExpiry, &u.Pronouns, &u.Socials, &u.MusicID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	if r.db == nil {
		return nil, errors.New("database connection is offline")
	}
	return scanUser(r.db.QueryRow("SELECT "+userSelectFields+" FROM users WHERE username = ?", username))
}

func (r *UserRepository) GetByID(id int) (*models.User, error) {
	if r.db == nil {
		return nil, errors.New("database connection is offline")
	}
	return scanUser(r.db.QueryRow("SELECT "+userSelectFields+" FROM users WHERE id = ?", id))
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	if r.db == nil {
		return nil, errors.New("database connection is offline")
	}
	return scanUser(r.db.QueryRow("SELECT "+userSelectFields+" FROM users WHERE mail = ?", email))
}

func (r *UserRepository) GetByUnikey(unikey string) (*models.User, error) {
	if r.db == nil {
		return nil, errors.New("database connection is offline")
	}
	return scanUser(r.db.QueryRow("SELECT "+userSelectFields+" FROM users WHERE unikey = ?", unikey))
}

func (r *UserRepository) SearchAdminUsers(search string, powerFilter int, limit, offset int) ([]*models.User, int, error) {
	if r.db == nil {
		return nil, 0, errors.New("database connection is offline")
	}

	search = strings.TrimSpace(search)

	var whereClauses []string
	var args []any

	if search != "" {
		searchPattern := "%" + search + "%"
		if id, err := strconv.Atoi(search); err == nil {
			whereClauses = append(whereClauses, "(username LIKE ? OR displayname LIKE ? OR mail LIKE ? OR id = ?)")
			args = append(args, searchPattern, searchPattern, searchPattern, id)
		} else {
			whereClauses = append(whereClauses, "(username LIKE ? OR displayname LIKE ? OR mail LIKE ?)")
			args = append(args, searchPattern, searchPattern, searchPattern)
		}
	}

	if powerFilter >= 0 {
		whereClauses = append(whereClauses, "power = ?")
		args = append(args, powerFilter)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM users " + whereSQL
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, limit, offset)

	query := "SELECT " + userSelectFields + " FROM users " + whereSQL + " ORDER BY id ASC LIMIT ? OFFSET ?"

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, nil
}

func (r *UserRepository) SearchUsers(query string, limit int) ([]*models.User, error) {
	if r.db == nil {
		return []*models.User{}, nil
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return []*models.User{}, nil
	}

	if limit <= 0 || limit > 50 {
		limit = 10
	}

	pattern := "%" + query + "%"
	q := "SELECT " + userSelectFields + " FROM users WHERE (username LIKE ? OR displayname LIKE ?) ORDER BY CASE WHEN username = ? THEN 0 WHEN username LIKE ? THEN 1 ELSE 2 END, id ASC LIMIT ?"
	rows, err := r.db.Query(q, pattern, pattern, query, query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepository) IsOldUsername(username string) (bool, error) {
	if r.db == nil {
		return false, nil
	}
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM oldusernames WHERE ousername = ?)", username).Scan(&exists)
	return exists, err
}

func (r *UserRepository) GetOldUsernames(userID, limit, offset int) ([]models.UsernameHistory, int, error) {
	if r.db == nil {
		return []models.UsernameHistory{}, 0, nil
	}
	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM oldusernames WHERE uid = ?", userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(`SELECT ousername, COALESCE(UNIX_TIMESTAMP(creation_date), 0)
		FROM oldusernames WHERE uid = ? ORDER BY creation_date DESC, id DESC LIMIT ? OFFSET ?`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	usernames := make([]models.UsernameHistory, 0)
	for rows.Next() {
		var entry models.UsernameHistory
		if err := rows.Scan(&entry.Username, &entry.ClaimedAt); err != nil {
			return nil, 0, err
		}
		usernames = append(usernames, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return usernames, total, nil
}

func (r *UserRepository) IsOldUsernameExceptUser(username string, userID int) (bool, error) {
	if r.db == nil {
		return false, nil
	}
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM oldusernames WHERE ousername = ? AND uid != ?)", username, userID).Scan(&exists)
	return exists, err
}

func (r *UserRepository) GetUsernameChangesLeft(userID int) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM oldusernames WHERE uid = ? AND creation_date > NOW() - INTERVAL 3 DAY", userID).Scan(&count)
	if err != nil {
		return 2, nil
	}
	left := 2 - count
	if left < 0 {
		left = 0
	}
	return left, nil
}

func (r *UserRepository) CreateUser(u *models.User, torsoColor, legColor string) error {
	if r.db == nil {
		return nil
	}
	query := "INSERT INTO users (username, displayname, mail, password, description, unikey, power, primary_clan, namecolor, custom_css, vermail, vermc, casom, bits, bucks, last_online, creation_date, views, feed_captchas, pronouns, socials) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	res, err := r.db.Exec(query,
		u.Username, u.DisplayName, u.Mail, u.Password, u.Description,
		u.Unikey, u.Power, u.PrimaryClan, u.NameColor, u.CustomCSS,
		u.Vermail, u.Vermc, u.Casom, u.Bits, u.Bucks,
		u.LastOnline, u.CreationDate, u.Views, u.FeedCaptchas, u.Pronouns, u.Socials,
	)
	if err != nil {
		return err
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = int(lastID)

	avatarQuery := "INSERT INTO avatar (id, head_color, larm_color, rarm_color, torso_color, lleg_color, rleg_color, hat1, hat2, hat3, hat4, hat5, tool, shirt, tshirt, pants, face, head, larm, rarm, torso, lleg, rleg, light_color, light_intensity) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	_, err = r.db.Exec(avatarQuery,
		u.ID, models.DefaultHeadColor, models.DefaultHeadColor, models.DefaultHeadColor, torsoColor, legColor, legColor,
		0, 0, 0, 0, 0, 0, 0, 0, 0, models.DefaultFaceItemID, 0, 0, 0, 0, 0, 0, "ffffff", 100,
	)
	return err
}

func (r *UserRepository) SetEmailVerification(userID int, token string, expiry int64) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.Exec("UPDATE users SET email_verify_token = ?, email_verify_expiry = ? WHERE id = ?", token, expiry, userID)
	return err
}

func (r *UserRepository) MarkEmailVerified(userID int, email string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.Exec("UPDATE users SET vermail = ?, email_verify_token = NULL, email_verify_expiry = NULL WHERE id = ?", email, userID)
	return err
}

func (r *UserRepository) GetByEmailToken(token string) (*models.User, error) {
	if r.db == nil {
		return nil, errors.New("database offline")
	}
	u := &models.User{}
	err := r.db.QueryRow("SELECT "+userSelectFields+" FROM users WHERE email_verify_token = ?", token).Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.Mail, &u.Password, &u.Description,
		&u.Unikey, &u.Power, &u.PrimaryClan, &u.NameColor, &u.CustomCSS,
		&u.Vermail, &u.Vermc, &u.Casom, &u.Bits, &u.Bucks,
		&u.LastOnline, &u.CreationDate, &u.Views, &u.FeedCaptchas,
		&u.EmailVerifyToken, &u.EmailVerifyExpiry, &u.Pronouns, &u.Socials, &u.MusicID,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) UpdateLastOnline(userID int) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.Exec("UPDATE users SET last_online = ? WHERE id = ?", time.Now(), userID)
	return err
}

func (r *UserRepository) GetUserCount() (int, error) {
	if r.db == nil {
		return 0, nil
	}
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (r *UserRepository) GetRecentUsers(excludeUsername string, limit int) ([]*models.User, error) {
	if r.db == nil {
		return nil, nil
	}
	query := "SELECT id, username, displayname FROM users WHERE username != ? ORDER BY last_online DESC LIMIT ?"
	rows, err := r.db.Query(query, excludeUsername, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (r *UserRepository) GetUserSocialCounts(userID int) (int, int, int) {
	if r.db == nil {
		return 0, 0, 0
	}
	var friends, followers, following int
	_ = r.db.QueryRow("SELECT COUNT(*) FROM friends WHERE (uid = ? OR fid = ?) AND state = 'accepted'", userID, userID).Scan(&friends)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM followers WHERE following_id = ?", userID).Scan(&followers)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM followers WHERE follower_id = ?", userID).Scan(&following)
	return friends, followers, following
}

func (r *UserRepository) ChangeDisplayName(userID int, newDisplayName, oldDisplayName string) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	var count int
	_ = tx.QueryRow("SELECT COUNT(*) FROM olddisplaynames WHERE uid = ? AND creation_date > NOW() - INTERVAL 7 DAY", userID).Scan(&count)
	if count >= 3 {
		tx.Rollback()
		return errors.New("You can only change your display name 3 times every 7 days")
	}
	if _, err := tx.Exec("UPDATE users SET displayname = ? WHERE id = ?", newDisplayName, userID); err != nil {
		tx.Rollback()
		return err
	}
	_, _ = tx.Exec("INSERT INTO olddisplaynames (uid, odisplayname) VALUES (?, ?)", userID, oldDisplayName)
	return tx.Commit()
}

func (r *UserRepository) ChangeUsername(userID int, oldUsername, newUsername string, cost int) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	var uID, bucks int
	err = tx.QueryRow("SELECT id, bucks FROM users WHERE username = ? FOR UPDATE", oldUsername).Scan(&uID, &bucks)
	if err != nil {
		tx.Rollback()
		return errors.New("User not found")
	}
	if bucks < cost {
		tx.Rollback()
		return errors.New("You do not have enough Vertices (requires 100)")
	}
	var exists bool
	_ = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", newUsername).Scan(&exists)
	if exists {
		tx.Rollback()
		return errors.New("Username is already taken")
	}
	_ = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM oldusernames WHERE ousername = ?)", newUsername).Scan(&exists)
	if exists {
		tx.Rollback()
		return errors.New("Username is already taken")
	}
	var rateLimitCount int
	_ = tx.QueryRow("SELECT COUNT(*) FROM oldusernames WHERE uid = ? AND creation_date > NOW() - INTERVAL 3 DAY", uID).Scan(&rateLimitCount)
	if rateLimitCount >= 2 {
		tx.Rollback()
		return errors.New("You can only change your username 2 times every 3 days")
	}
	if _, err := tx.Exec("UPDATE users SET username = ?, bucks = bucks - ? WHERE id = ?", newUsername, cost, uID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec("INSERT INTO oldusernames (id, uid, ousername) VALUES ((SELECT COALESCE(MAX(x.id), 0) + 1 FROM (SELECT id FROM oldusernames) x), ?, ?)", uID, oldUsername); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *UserRepository) UpdatePassword(userID int, hashedPassword string) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	_, err := r.db.Exec("UPDATE users SET password = ? WHERE id = ?", hashedPassword, userID)
	return err
}

func (r *UserRepository) UpdateBio(userID int, bio string) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	_, err := r.db.Exec("UPDATE users SET description = ? WHERE id = ?", bio, userID)
	return err
}

func (r *UserRepository) UpdatePronouns(userID int, pronouns string) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	_, err := r.db.Exec("UPDATE users SET pronouns = ? WHERE id = ?", pronouns, userID)
	return err
}

func (r *UserRepository) UpdateMusicID(userID int, trackID int64) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	var val any
	if trackID > 0 {
		val = trackID
	}
	_, err := r.db.Exec("UPDATE users SET music_id = ? WHERE id = ?", val, userID)
	return err
}

func (r *UserRepository) AdminSetUsername(userID int, username string) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	_, err := r.db.Exec("UPDATE users SET username = ? WHERE id = ?", username, userID)
	return err
}

func (r *UserRepository) AdminSetDisplayName(userID int, displayName string) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	_, err := r.db.Exec("UPDATE users SET displayname = ? WHERE id = ?", displayName, userID)
	return err
}

func (r *UserRepository) UpdateSocials(userID int, socialsJSON string) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	_, err := r.db.Exec("UPDATE users SET socials = ? WHERE id = ?", socialsJSON, userID)
	return err
}

func (r *UserRepository) CreateRememberToken(userID int, tokenHash string, expiresAt time.Time) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	_, err := r.db.Exec("INSERT INTO remember_tokens (uid, thash, exp) VALUES (?, ?, ?)", userID, tokenHash, expiresAt)
	return err
}

func (r *UserRepository) GetRememberTokenExpiry(userID int, tokenHash string) (time.Time, error) {
	if r.db == nil {
		return time.Time{}, errors.New("database connection is offline")
	}
	var exp time.Time
	err := r.db.QueryRow("SELECT exp FROM remember_tokens WHERE uid = ? AND thash = ?", userID, tokenHash).Scan(&exp)
	return exp, err
}

func (r *UserRepository) DeleteRememberToken(tokenHash string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.Exec("DELETE FROM remember_tokens WHERE thash = ?", tokenHash)
	return err
}

func (r *UserRepository) UserExists(userID int) (bool, error) {
	if r.db == nil {
		return false, errors.New("database connection is offline")
	}
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
	return exists, err
}
