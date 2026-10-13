package repository

import (
	"database/sql"

	"buildoverse/backend/models"
)

type FeedRepository struct {
	db *sql.DB
}

func NewFeedRepository(db *sql.DB) *FeedRepository {
	return &FeedRepository{db: db}
}

func scanFeedPost(rows *sql.Rows) (*models.FeedPost, error) {
	var p models.FeedPost
	if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Content, &p.Removed, &p.Edited, &p.EditDate, &p.CreationDate, &p.Reactions, &p.HasReacted); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *FeedRepository) queryFeedPosts(query string, args ...any) ([]*models.FeedPost, error) {
	if r.db == nil {
		return nil, nil
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*models.FeedPost
	for rows.Next() {
		p, err := scanFeedPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func (r *FeedRepository) checkExists(query string, args ...any) (bool, error) {
	if r.db == nil {
		return false, nil
	}
	var exists bool
	err := r.db.QueryRow(query, args...).Scan(&exists)
	return exists, err
}

func (r *FeedRepository) execQuery(query string, args ...any) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.Exec(query, args...)
	return err
}

func (r *FeedRepository) getCount(query string, args ...any) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (r *FeedRepository) GetRecentFeedPosts(limit, currentUserID int) ([]*models.FeedPost, error) {
	return r.GetRecentFeedPostsPaginated(limit, 0, currentUserID)
}

func (r *FeedRepository) GetRecentFeedPostsPaginated(limit, offset, currentUserID int) ([]*models.FeedPost, error) {
	query := `SELECT f.id, f.user_id, u.username, f.status, f.removed, f.edited, f.edit_date, f.creation_date,
	                 (SELECT COUNT(*) FROM freact r WHERE r.fid = f.id) AS reactions,
	                 EXISTS(SELECT 1 FROM freact r WHERE r.fid = f.id AND r.uid = ?) AS has_reacted
              FROM feed f 
              INNER JOIN users u ON f.user_id = u.id 
              WHERE f.removed = 'false' 
              ORDER BY f.creation_date DESC, f.id DESC LIMIT ? OFFSET ?`
	return r.queryFeedPosts(query, currentUserID, limit, offset)
}

func (r *FeedRepository) CreatePost(userID int, content string) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	_, err := r.db.Exec("INSERT INTO feed (id, user_id, status) VALUES ((SELECT COALESCE(MAX(x.id), 0) + 1 FROM (SELECT id FROM feed) x), ?, ?)", userID, content)
	if err != nil {
		return 0, err
	}
	var postID int
	err = r.db.QueryRow("SELECT MAX(id) FROM feed WHERE user_id = ?", userID).Scan(&postID)
	return postID, err
}

func (r *FeedRepository) HasUserReacted(userID, feedID int) (bool, error) {
	return r.checkExists("SELECT EXISTS(SELECT 1 FROM freact WHERE uid = ? AND fid = ?)", userID, feedID)
}

func (r *FeedRepository) AddReaction(userID, feedID int) error {
	return r.execQuery("INSERT INTO freact (uid, fid) VALUES (?, ?)", userID, feedID)
}

func (r *FeedRepository) RemoveReaction(userID, feedID int) error {
	return r.execQuery("DELETE FROM freact WHERE uid = ? AND fid = ?", userID, feedID)
}

func (r *FeedRepository) GetReactionCount(feedID int) (int, error) {
	return r.getCount("SELECT COUNT(*) FROM freact WHERE fid = ?", feedID)
}

func (r *FeedRepository) GetRecentFriendsFeedPosts(limit, currentUserID int) ([]*models.FeedPost, error) {
	return r.GetRecentFriendsFeedPostsPaginated(limit, 0, currentUserID)
}

func (r *FeedRepository) GetRecentFriendsFeedPostsPaginated(limit, offset, currentUserID int) ([]*models.FeedPost, error) {
	query := `SELECT f.id, f.userid, u.username, f.status, f.removed, f.edited, f.editdate, f.creationdate,
	                 (SELECT COUNT(*) FROM ffreact r WHERE r.fid = f.id) AS reactions,
	                 EXISTS(SELECT 1 FROM ffreact r WHERE r.fid = f.id AND r.uid = ?) AS has_reacted
              FROM ffeed f 
              INNER JOIN users u ON f.userid = u.id 
              WHERE f.removed = 'false' 
                AND (f.userid = ? OR f.userid IN (
                    SELECT CASE WHEN uid = ? THEN fid ELSE uid END 
                    FROM friends 
                    WHERE (uid = ? OR fid = ?) AND state = 'accepted'
                ))
              ORDER BY f.creationdate DESC, f.id DESC LIMIT ? OFFSET ?`
	return r.queryFeedPosts(query, currentUserID, currentUserID, currentUserID, currentUserID, currentUserID, limit, offset)
}

func (r *FeedRepository) CreateFriendsPost(userID int, content string) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	_, err := r.db.Exec("INSERT INTO ffeed (id, userid, status) VALUES ((SELECT COALESCE(MAX(x.id), 0) + 1 FROM (SELECT id FROM ffeed) x), ?, ?)", userID, content)
	if err != nil {
		return 0, err
	}
	var postID int
	err = r.db.QueryRow("SELECT MAX(id) FROM ffeed WHERE userid = ?", userID).Scan(&postID)
	return postID, err
}

func (r *FeedRepository) HasUserReactedFriends(userID, feedID int) (bool, error) {
	return r.checkExists("SELECT EXISTS(SELECT 1 FROM ffreact WHERE uid = ? AND fid = ?)", userID, feedID)
}

func (r *FeedRepository) AddReactionFriends(userID, feedID int) error {
	return r.execQuery("INSERT INTO ffreact (uid, fid) VALUES (?, ?)", userID, feedID)
}

func (r *FeedRepository) RemoveReactionFriends(userID, feedID int) error {
	return r.execQuery("DELETE FROM ffreact WHERE uid = ? AND fid = ?", userID, feedID)
}

func (r *FeedRepository) GetReactionCountFriends(feedID int) (int, error) {
	return r.getCount("SELECT COUNT(*) FROM ffreact WHERE fid = ?", feedID)
}

func (r *FeedRepository) CreateComment(feedID, userID int, parentID *int, feedType, comment string) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	var parentVal sql.NullInt64
	if parentID != nil && *parentID > 0 {
		parentVal = sql.NullInt64{Int64: int64(*parentID), Valid: true}
	}
	query := `INSERT INTO feed_comments (id, feed_id, user_id, parent_id, feed_type, comment) 
	          VALUES ((SELECT COALESCE(MAX(x.id), 0) + 1 FROM (SELECT id FROM feed_comments) x), ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, feedID, userID, parentVal, feedType, comment)
	if err != nil {
		return 0, err
	}
	var commentID int
	err = r.db.QueryRow("SELECT MAX(id) FROM feed_comments WHERE user_id = ?", userID).Scan(&commentID)
	return commentID, err
}

func (r *FeedRepository) GetCommentsByFeedID(feedID int, feedType string, currentUserID int) ([]*models.FeedComment, error) {
	if r.db == nil {
		return nil, nil
	}
	query := `SELECT c.id, c.feed_id, c.user_id, c.parent_id, c.feed_type, u.username, c.comment, c.removed, c.edited, c.creation_date,
	                 (SELECT COUNT(*) FROM creact r WHERE r.cid = c.id) AS reactions,
	                 EXISTS(SELECT 1 FROM creact r WHERE r.cid = c.id AND r.uid = ?) AS has_reacted
              FROM feed_comments c
              INNER JOIN users u ON c.user_id = u.id
              WHERE c.feed_id = ? AND c.feed_type = ? AND c.removed = 'false'
              ORDER BY c.creation_date DESC, c.id DESC`
	rows, err := r.db.Query(query, currentUserID, feedID, feedType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*models.FeedComment
	for rows.Next() {
		var c models.FeedComment
		if err := rows.Scan(&c.ID, &c.FeedID, &c.UserID, &c.ParentID, &c.FeedType, &c.Username, &c.Comment, &c.Removed, &c.Edited, &c.CreationDate, &c.Reactions, &c.HasReacted); err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}
	return comments, nil
}

func (r *FeedRepository) HasUserReactedComment(userID, commentID int) (bool, error) {
	return r.checkExists("SELECT EXISTS(SELECT 1 FROM creact WHERE uid = ? AND cid = ?)", userID, commentID)
}

func (r *FeedRepository) AddCommentReaction(userID, commentID int) error {
	return r.execQuery("INSERT INTO creact (uid, cid) VALUES (?, ?)", userID, commentID)
}

func (r *FeedRepository) RemoveCommentReaction(userID, commentID int) error {
	return r.execQuery("DELETE FROM creact WHERE uid = ? AND cid = ?", userID, commentID)
}

func (r *FeedRepository) GetCommentReactionCount(commentID int) (int, error) {
	return r.getCount("SELECT COUNT(*) FROM creact WHERE cid = ?", commentID)
}