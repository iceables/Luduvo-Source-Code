package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"buildoverse/backend/models"
	"buildoverse/backend/service"
)

type FeedPost struct {
	ID         int
	Username   string
	UserID     int
	Content    string
	TimeAgo    string
	FullDate   string
	Reactions  int
	HasReacted bool
}

type wsMessage struct {
	Type      string `json:"type"`
	ID        int    `json:"id"`
	Username  string `json:"username"`
	UserID    int    `json:"user_id"`
	Content   string `json:"content"`
	TimeAgo   string `json:"time_ago"`
	FullDate  string `json:"full_date"`
	Reactions int    `json:"reactions"`
	FeedType  string `json:"feed_type"`
}

type WSClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type clientInfo struct {
	client   *WSClient
	username string
}

var (
	feedClients = make(map[*WSClient]string)
	feedHubMu   sync.Mutex
)

func getFeedClientsSnapshot() []clientInfo {
	feedHubMu.Lock()
	defer feedHubMu.Unlock()
	clients := make([]clientInfo, 0, len(feedClients))
	for client, uName := range feedClients {
		clients = append(clients, clientInfo{client: client, username: uName})
	}
	return clients
}

func toFeedPosts(dbPosts []*models.FeedPost) []FeedPost {
	posts := make([]FeedPost, 0, len(dbPosts))
	for _, dbP := range dbPosts {
		posts = append(posts, FeedPost{
			ID:         dbP.ID,
			Username:   dbP.Username,
			UserID:     dbP.UserID,
			Content:    dbP.Content,
			TimeAgo:    service.FormatTimeAgo(dbP.CreationDate),
			FullDate:   dbP.CreationDate.Format("January 02, 2006 at 03:04 PM"),
			Reactions:  dbP.Reactions,
			HasReacted: dbP.HasReacted,
		})
	}
	return posts
}

func sendFeedError(client *WSClient, err error) {
	errPayload, _ := json.Marshal(wsMessage{
		Type:    "error",
		Content: err.Error(),
	})
	_ = client.Write(websocket.TextMessage, errPayload)
}

func broadcastToClients(data []byte) {
	clients := getFeedClientsSnapshot()
	for _, item := range clients {
		_ = item.client.Write(websocket.TextMessage, data)
	}
}

func Home(c fiber.Ctx) error {
	var count int
	var games []*models.Game
	if service.User != nil {
		count, _ = service.User.GetUserCount()
	}
	if service.Game != nil {
		var err error
		games, err = service.Game.GetPopularGames(12)
		if err != nil {
			log.Printf("Buildoverse DB Error: Failed to fetch popular games: %v", err)
		}
	}

	username := GetActiveUser(c)
	var greeting string
	var friends []*models.User

	if username != "" {
		greetings := []string{
			"Hello, %s!",
			"G'day, %s!",
			"Welcome back, %s!",
			"Howdy, %s!",
			"What's cracking, %s!",
			"Salutations, %s!",
			"Yo, %s!",
			"Greetings, %s!",
			"Hey there, %s!",
			"Good to see you, %s!",
			"Hi, %s!",
			"Sup, %s!",
			"Ahoy, %s!",
			"What's up, %s!",
			"Aloha, %s!",
			"Nice to see you, %s!",
			"What's new, %s!",
			"Long time no see, %s!",
			"Look who it is, %s!",
			"Welcome, %s!",
		}
		idx := time.Now().UnixNano() % int64(len(greetings))
		greeting = fmt.Sprintf(greetings[idx], username)

		if service.User != nil {
			friends, _ = service.User.GetRecentUsers(username, 10)
		}
	}

	var currentUserID int
	if username != "" && service.User != nil {
		u, _ := service.User.GetUserByUsername(username)
		if u != nil {
			currentUserID = u.ID
		}
	}

	var currentFeed []FeedPost
	if service.Feed != nil {
		dbPosts, err := service.Feed.GetRecentFeed(10, currentUserID, "worldwide")
		if err == nil {
			currentFeed = toFeedPosts(dbPosts)
		}
	}

	signupMessage := ""
	if sess := session.FromContext(c); sess != nil {
		if raw := sess.Get("signup_message"); raw != nil {
			signupMessage, _ = raw.(string)
			sess.Delete("signup_message")
		}
	}

	return Render(c, "pages/home", fiber.Map{
		"Title":         "Buildoverse",
		"UserCount":     count,
		"Games":         games,
		"Greeting":      greeting,
		"Friends":       friends,
		"Feed":          currentFeed,
		"SignupMessage": signupMessage,
	}, "layouts/main")
}

func GetFeedPaginated(c fiber.Ctx) error {
	limitStr := c.Query("limit", "10")
	offsetStr := c.Query("offset", "0")
	feedType := c.Query("type", "worldwide")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	username := GetActiveUser(c)
	var currentUserID int
	if username != "" && service.User != nil {
		u, _ := service.User.GetUserByUsername(username)
		if u != nil {
			currentUserID = u.ID
		}
	}

	if service.Feed == nil {
		return c.JSON([]FeedPost{})
	}

	dbPosts, err := service.Feed.GetRecentFeedPaginated(limit, offset, currentUserID, feedType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(toFeedPosts(dbPosts))
}

func GetFeedCommentsHandler(c fiber.Ctx) error {
	feedID, err := strconv.Atoi(c.Query("feed_id"))
	if err != nil || feedID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid feed_id"})
	}
	feedType := c.Query("feed_type", "worldwide")
	username := GetActiveUser(c)

	if service.Feed == nil {
		return c.JSON([]*models.FeedComment{})
	}

	comments, err := service.Feed.GetFeedComments(feedID, feedType, username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(comments)
}

func PostFeed(c fiber.Ctx) error {
	username := GetActiveUser(c)
	if username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	content := c.FormValue("content")
	feedType := c.FormValue("feed_type", "worldwide")
	if service.Feed == nil {
		return c.Redirect().To("/")
	}

	postID, userID, err := service.Feed.PostToFeed(username, content, feedType)
	if err != nil {
		return c.Redirect().To("/")
	}

	if postID > 0 {
		BroadcastFeedPost(postID, username, userID, content, feedType)
	}

	return c.Redirect().To("/")
}

func FeedWS(c *websocket.Conn) {
	usernameVal := c.Locals("username")
	username, ok := usernameVal.(string)
	if !ok || username == "" {
		_ = c.Close()
		return
	}

	client := &WSClient{conn: c}

	feedHubMu.Lock()
	feedClients[client] = username
	feedHubMu.Unlock()

	defer func() {
		feedHubMu.Lock()
		delete(feedClients, client)
		feedHubMu.Unlock()
		_ = c.Close()
	}()

	type WSClientMessage struct {
		Type     string `json:"type"`
		Content  string `json:"content"`
		FID      int    `json:"fid"`
		CID      int    `json:"cid"`
		FeedID   int    `json:"feed_id"`
		ParentID *int   `json:"parent_id"`
		FeedType string `json:"feed_type"`
	}

	for {
		mt, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		if mt == websocket.TextMessage {
			var clientMsg WSClientMessage
			if err := json.Unmarshal(msg, &clientMsg); err == nil {
				feedType := clientMsg.FeedType
				if feedType == "" {
					feedType = "worldwide"
				}

				switch clientMsg.Type {
				case "react":
					if clientMsg.FID > 0 && service.Feed != nil {
						totalReactions, err := service.Feed.ToggleReaction(username, clientMsg.FID, feedType)
						if err == nil {
							BroadcastReactionUpdate(clientMsg.FID, totalReactions, feedType)
						} else {
							sendFeedError(client, err)
						}
					}

				case "creact":
					if clientMsg.CID > 0 && service.Feed != nil {
						totalReactions, err := service.Feed.ToggleCommentReaction(username, clientMsg.CID)
						if err == nil {
							BroadcastCommentReactionUpdate(clientMsg.CID, totalReactions, feedType)
						} else {
							sendFeedError(client, err)
						}
					}

				case "comment":
					if clientMsg.FeedID > 0 && service.Feed != nil {
						comment, err := service.Feed.PostComment(username, clientMsg.FeedID, clientMsg.ParentID, feedType, clientMsg.Content)
						if err != nil {
							sendFeedError(client, err)
							continue
						}
						BroadcastCommentPost(comment)
					}

				default:
					if service.Feed != nil {
						postID, _, err := service.Feed.PostToFeed(username, clientMsg.Content, feedType)
						if err != nil {
							sendFeedError(client, err)
							continue
						}

						if postID > 0 {
							u, _ := service.User.GetUserByUsername(username)
							uID := 0
							if u != nil {
								uID = u.ID
							}
							BroadcastFeedPost(postID, username, uID, clientMsg.Content, feedType)
						}
					}
				}
			}
		}
	}
}

func (ws *WSClient) Write(messageType int, data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.conn.WriteMessage(messageType, data)
}

func BroadcastFeedPost(id int, username string, userID int, content string, feedType string) {
	msg := wsMessage{
		Type:      "new_post",
		ID:        id,
		Username:  username,
		UserID:    userID,
		Content:   content,
		TimeAgo:   "Just now",
		FullDate:  time.Now().Format("January 02, 2006 at 03:04 PM"),
		Reactions: 0,
		FeedType:  feedType,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	clients := getFeedClientsSnapshot()

	for _, item := range clients {
		if feedType == "friends" {
			senderUser, _ := service.User.GetUserByID(userID)
			recipientUser, _ := service.User.GetUserByUsername(item.username)
			if senderUser != nil && recipientUser != nil {
				if senderUser.ID != recipientUser.ID {
					status, _ := service.Friend.GetFriendStatus(recipientUser.ID, senderUser.ID)
					if status != "friends" {
						continue
					}
				}
			}
		}
		_ = item.client.Write(websocket.TextMessage, data)
	}
}

func BroadcastCommentPost(comment *models.FeedComment) {
	msg := struct {
		Type    string              `json:"type"`
		Comment *models.FeedComment `json:"comment"`
	}{
		Type:    "new_comment",
		Comment: comment,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	broadcastToClients(data)
}

func BroadcastReactionUpdate(fid int, reactions int, feedType string) {
	msg := struct {
		Type       string `json:"type"`
		FID        int    `json:"fid"`
		Reactions  int    `json:"reactions"`
		HasReacted bool   `json:"has_reacted"`
		FeedType   string `json:"feed_type"`
	}{
		Type:      "reaction_update",
		FID:       fid,
		Reactions: reactions,
		FeedType:  feedType,
	}

	clients := getFeedClientsSnapshot()

	for _, item := range clients {
		hasReacted := false
		if service.Feed != nil {
			hasReacted = service.Feed.HasUserReacted(item.username, fid, feedType)
		}
		msg.HasReacted = hasReacted

		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		_ = item.client.Write(websocket.TextMessage, data)
	}
}

func BroadcastCommentReactionUpdate(cid int, reactions int, feedType string) {
	msg := struct {
		Type       string `json:"type"`
		CID        int    `json:"cid"`
		Reactions  int    `json:"reactions"`
		HasReacted bool   `json:"has_reacted"`
		FeedType   string `json:"feed_type"`
	}{
		Type:      "comment_reaction_update",
		CID:       cid,
		Reactions: reactions,
		FeedType:  feedType,
	}

	clients := getFeedClientsSnapshot()

	for _, item := range clients {
		hasReacted := false
		if service.Feed != nil {
			hasReacted = service.Feed.HasUserReactedComment(item.username, cid)
		}
		msg.HasReacted = hasReacted

		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		_ = item.client.Write(websocket.TextMessage, data)
	}
}