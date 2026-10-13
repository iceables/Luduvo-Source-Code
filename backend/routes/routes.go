package routes

import (
	"strings"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"buildoverse/backend/config"
	"buildoverse/backend/handlers"
)

func Setup(app *fiber.App) {
	app.Use(func(c fiber.Ctx) error {
		if config.Global != nil && config.Global.IsLiveTimerActive() {
			path := c.Path()
			if path != "/livetimer" &&
				path != "/api/v1/livetimer" &&
				!strings.HasPrefix(path, "/static") &&
				!strings.HasPrefix(path, "/assets") &&
				path != "/favicon.ico" {
				if c.Get("HX-Request") == "true" {
					c.Set("HX-Redirect", "/livetimer")
					return c.SendStatus(fiber.StatusFound)
				}
				return c.Redirect().To("/livetimer")
			}
		}
		return c.Next()
	})

	// ============================================================
	// LUDUVO-COMPATIBLE ROUTE LAYER
	// All routes match luduvo.com URL structure
	// ============================================================

	// --- Altcha / Security ---
	app.Get("/altcha", handlers.AltchaGet)
	app.Get("/api/v1/altcha", handlers.AltchaGet)

	// --- Auth Routes (luduvo paths) ---
	app.Get("/auth", handlers.AuthIndex)
	app.Get("/auth/login", handlers.LoginGet)
	app.Get("/login", handlers.LoginGet)
	app.Post("/auth/login", handlers.LoginPostAPI)
	app.Post("/login", handlers.LoginPost)
	app.Post("/auth/logout", handlers.LogoutAPI)
	app.Get("/auth/register", handlers.RegisterGet)
	app.Get("/register", handlers.RegisterGet)
	app.Post("/auth/register", handlers.RegisterPostAPI)
	app.Post("/register", handlers.RegisterPost)
	app.Post("/auth/refresh", handlers.AuthRefresh)
	app.Post("/auth/forgot-password", handlers.AuthForgotPassword)
	app.Post("/auth/reset-password", handlers.AuthResetPassword)
	app.Get("/auth/verify-email", handlers.AuthVerifyEmail)
	app.Post("/auth/verify-email", handlers.AuthVerifyEmail)
	app.Post("/auth/resend-verification", handlers.AuthResendVerification)
	app.Get("/auth/ticket", handlers.AuthTicket)
	app.Post("/auth/ticket", handlers.AuthTicket)
	app.Get("/auth/confirm-email-change", handlers.AuthConfirmEmailChange)
	app.Get("/auth/studio", handlers.AuthStudio)

	// Gameserver auth
	app.Get("/auth/validate", handlers.ValidateUkey)
	app.Get("/api/v1/auth/validate", handlers.ValidateUkey)

	// --- Home / Main Pages ---
	app.Get("/heartbeat", handlers.Heartbeat)
	app.Get("/api/v1/heartbeat", handlers.Heartbeat)
	app.Get("/play", handlers.PlayPage)
	app.Get("/create", handlers.CreateGet)
	app.Get("/clans", handlers.ClansPage)
	app.Get("/forum", handlers.ForumPage)
	app.Get("/currency", handlers.CurrencyPage)
	app.Get("/redeem", handlers.RedeemPage)
	app.Get("/tos", handlers.LegalPage)
	app.Get("/logout", handlers.Logout)
	app.Get("/", handlers.Home)
	app.Get("/home", handlers.Home)

	// --- Search ---
	app.Get("/search", handlers.SearchPage)
	app.Get("/api/v1/search", handlers.SearchAPI)

	// --- User Profile Routes (luduvo-style) ---
	app.Get("/users", handlers.UsersSearch)
	app.Get("/users/search", handlers.UsersSearch)
	app.Get("/users/by-username/:username/profile", handlers.UserByUsernameProfile)
	app.Get("/users/:id/usernames", handlers.UsernamesHistory)
	app.Get("/users/:id/profile", handlers.UserProfileAPI)
	app.Get("/u/:id", handlers.UserProfilePage)
	app.Get("/user/:id", handlers.UserProfilePage)
	app.Get("/users/:id/avatar/headshot", handlers.AvatarHeadshotGet)
	app.Get("/users/:id/avatar/full-body", handlers.UserAvatarFullBody)
	app.Get("/users/:id/presence", handlers.UserPresence)
	app.Get("/users/:id/inventory", handlers.UserInventory)
	app.Get("/users/:id/places", handlers.UserPlaces)
	app.Get("/users/:id/friends", handlers.UserFriends)
	app.Get("/users/:id/groups", handlers.UserGroups)
	app.Get("/users/presence", handlers.UsersPresence)
	app.Post("/users/:id/join", handlers.UserJoinPlace)

	// --- Me / Current User Routes ---
	app.Get("/me", handlers.MeHandler)
	app.Get("/me/profile", handlers.MeProfile)
	app.Get("/me/settings", handlers.SettingsGetCompat)
	app.Put("/me/settings", handlers.SettingsPutCompat)
	app.Post("/me/settings", handlers.SettingsPost)
	app.Get("/me/inventory", handlers.MeInventory)
	app.Get("/me/ignores", handlers.MeIgnoredList)
	app.Post("/me/ignores/:id", handlers.AddIgnoredUser)
	app.Delete("/me/ignores/:id", handlers.RemoveIgnoredUser)
	app.Get("/me/blocks", handlers.MeBlockedList)
	app.Post("/me/blocks/:id", handlers.AddBlockedUser)
	app.Delete("/me/blocks/:id", handlers.RemoveBlockedUser)
	app.Get("/me/balance", handlers.MeBalance)
	app.Get("/me/bootstrap", handlers.MeBootstrap)
	app.Get("/me/friends", handlers.MeFriends)
	app.Get("/me/friends/pending", handlers.MeFriendsPending)
	app.Get("/me/places", handlers.MePlaces)
	app.Get("/me/groups", handlers.MeGroups)
	app.Get("/me/recently-played", handlers.RecentlyPlayed)
	app.Get("/me/deletions", handlers.MeDeletions)
	app.Get("/me/enforcement", handlers.MeEnforcement)
	app.Get("/me/appeals", handlers.MeAppeals)
	app.Get("/me/appeals/:id", handlers.MeAppealDetail)
	app.Get("/me/appeals/:id/messages", handlers.MeAppealMessages)
	app.Get("/me/suspension/acknowledge", handlers.MeSuspensionAcknowledge)
	app.Get("/me/suspension/pending-acknowledgment", handlers.MeSuspensionPending)
	app.Post("/me/game-moderation/acknowledge", handlers.MeGameModerationAck)
	app.Post("/me/group-moderation/acknowledge", handlers.MeGroupModerationAck)
	app.Post("/me/item-moderation/acknowledge", handlers.MeItemModerationAck)
	app.Post("/me/legal/accept", handlers.MeLegalAccept)
	app.Get("/me/mfa/status", handlers.MFAStatus)
	app.Post("/me/mfa/setup", handlers.MFASetup)
	app.Post("/me/mfa/verify", handlers.MFAVerify)
	app.Post("/me/mfa/disable", handlers.MFADisable)
	app.Get("/me/mfa/recovery-codes/regenerate", handlers.MFARecoveryCodes)
	app.Get("/me/official-messages/read", handlers.MeOfficialMessagesRead)
	app.Get("/me/outfits", handlers.MeOutfits)
	app.Post("/me/outfits", handlers.MeOutfitsCreate)
	app.Put("/me/outfits/:id", handlers.MeOutfitRename)
	app.Delete("/me/outfits/:id", handlers.MeOutfitDelete)
	app.Post("/me/outfits/:id/wear", handlers.MeOutfitWear)
	app.Get("/me/equipped", handlers.MeEquipped)
	app.Put("/me/equipped", handlers.MeEquippedPut)
	app.Get("/me/bundle", handlers.MeBundle)
	app.Get("/me/subscription", handlers.MeSubscription)
	app.Post("/me/subscription/cancel", handlers.MeSubscriptionCancel)
	app.Get("/me/pending-payments", handlers.MePendingPayments)
	app.Post("/me/checkout", handlers.MeCheckout)
	app.Get("/notifications", handlers.NotificationsList)
	app.Get("/conversations", handlers.ConversationsList)
	app.Get("/trades", handlers.TradesList)
	app.Post("/auth/login/mfa", handlers.AuthLoginMFA)
	app.Post("/auth/logout-all", handlers.AuthLogoutAll)
	app.Get("/me/rewards/daily", handlers.MeDailyRewards)
	app.Get("/me/transfers", handlers.MeTransfers)
	app.Get("/me/trades/counts", handlers.MeTradesCounts)
	app.Get("/me/profile/background", handlers.MeProfileBackground)
	app.Get("/me/profile/banner", handlers.MeProfileBanner)
	app.Get("/me/profile/picture", handlers.MeProfilePicture)
	app.Post("/me/publish", handlers.MePublish)
	app.Get("/me/ignores/:id", handlers.MeIgnores)
	app.Get("/me/warnings/:id/acknowledge", handlers.MeWarningsAcknowledge)
	app.Get("/me/reports/:id", handlers.MeReportsDetail)
	app.Get("/me/reports", handlers.MyReports)
	app.Get("/me/tickets/:id", handlers.MeTicketsDetail)
	app.Get("/me/creator-blacklist/acknowledge", handlers.MeCreatorBlacklistAck)
	app.Get("/me/places/:id/age-questionnaire", handlers.MePlacesAgeQuestionnaire)
	app.Get("/me/places/:id/rating-review", handlers.MePlacesRatingReview)

	// --- Friends Routes (luduvo paths) ---
	friends := app.Group("/friends")
	friends.Get("/", handlers.FriendsGetAPI)
	friends.Get("/:id", handlers.FriendDetail)
	friends.Post("/by-username/:username/request", handlers.FriendByUsernameRequest)
	friends.Post("/send/:id", handlers.FriendSendPostAPI)
	friends.Post("/accept/:id", handlers.FriendAcceptPostAPI)
	friends.Post("/decline/:id", handlers.FriendDeclinePostAPI)
	friends.Post("/cancel/:id", handlers.FriendCancelPostAPI)
	friends.Post("/remove/:id", handlers.FriendRemovePostAPI)
	friends.Post("/:id/block", handlers.FriendBlock)
	friends.Post("/:id/unblock", handlers.FriendUnblock)
	friends.Post("/:id/block", handlers.FriendBlockPost)
	friends.Post("/:id/unblock", handlers.FriendUnblockPost)
	friends.Post("/:id/accept", handlers.FriendAcceptPostAPI)
	friends.Delete("/:id", handlers.FriendRemovePostAPI)

	// --- Places / Games Routes (luduvo-style, maps to Buildoverse games) ---
	app.Get("/site/banners", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"banners": []any{}})
	})
	app.Get("/api/v1/store", handlers.CatalogStore)
	app.Get("/model-asset", handlers.CatalogModelAsset)
	app.Get("/store/avatar-rules", handlers.StoreAvatarRules)
	app.Get("/store/categories", handlers.ShopCategories)
	app.Get("/places", handlers.PlacesList)
	app.Get("/places/discovery", handlers.PlacesDiscovery)
	app.Get("/thumbnails/:id.png", handlers.GameThumbnail)
	app.Get("/places/:id", handlers.PlaceDetail)
	app.Get("/places/:id/servers", handlers.PlaceServers)
	app.Post("/places/:id/join", handlers.PlaceJoin)
	app.Get("/places/:id/rating", handlers.PlaceRating)
	app.Post("/places/:id/rate", handlers.PlaceRate)
	app.Get("/places/:id/world", handlers.PlaceWorld)
	app.Get("/places/:id/thumbnail", handlers.PlaceThumbnail)
	app.Get("/places/:id/up-next", handlers.PlaceUpNext)
	app.Get("/places/:id/bans/:uid", handlers.PlaceBans)
	app.Get("/places/:id/problems", handlers.PlaceProblems)
	app.Get("/places/:id/products/:pid/purchase", handlers.PlaceProductPurchase)
	app.Post("/places/:id/paid-access/purchase", handlers.PlacePaidAccessPurchase)
	app.Get("/places/:id/transfers", handlers.PlaceTransfers)
	app.Get("/places/transfers/:id/accept", handlers.PlaceTransferAccept)
	app.Get("/places/transfers/:id/reject", handlers.PlaceTransferReject)
	app.Get("/api/discover/rows/:id", handlers.ApiDiscoverRows)

	// Create places
	app.Get("/create/places", handlers.CreatePlacesPage)
	app.Get("/create/places/", handlers.CreatePlacesPage)

	// --- Games ---
	app.Get("/games", handlers.GamesPage)

	// --- Groups Routes (luduvo paths) ---
	app.Get("/groups", handlers.GroupsPage)
	app.Get("/groups/browse", handlers.GroupsBrowse)
	app.Get("/groups/search", handlers.GroupsAPI)
	app.Post("/groups", handlers.GroupsCreate)
	app.Get("/groups/:id", handlers.GroupDetail)
	app.Get("/groups/:id/places", handlers.GroupPlaces)
	app.Get("/groups/:id/announcements", handlers.GroupAnnouncements)
	app.Get("/groups/:id/announcements/:aid", handlers.GroupAnnouncementDetail)
	app.Post("/groups/:id/announcements/:aid/reactions", handlers.GroupAnnouncementReaction)
	app.Get("/groups/:id/announcements/:aid/replies", handlers.GroupAnnouncementReplies)
	app.Get("/groups/:id/announcements/:aid/replies/:rid", handlers.GroupAnnouncementReplyDetail)
	app.Get("/groups/:id/roles", handlers.GroupRoles)
	app.Get("/groups/:id/roles/:rid", handlers.GroupRoleDetail)
	app.Post("/groups/:id/roles/reorder", handlers.GroupRolesReorder)
	app.Get("/groups/:id/members/:uid/roles/:rid", handlers.GroupMemberRole)
	app.Get("/groups/:id/invites", handlers.GroupInvites)
	app.Post("/groups/:id/join", handlers.GroupJoin)
	app.Post("/groups/:id/leave", handlers.GroupLeave)
	app.Post("/groups/:id/kick/:uid", handlers.GroupKick)
	app.Get("/groups/:id/bans/:uid", handlers.GroupBans)
	app.Post("/groups/:id/ban/:uid", handlers.GroupBan)
	app.Get("/groups/:id/wall", handlers.GroupWall)
	app.Get("/groups/:id/wall/:pid", handlers.GroupWallPost)
	app.Get("/groups/:id/payouts", handlers.GroupPayouts)
	app.Get("/groups/:id/store/:itemid", handlers.GroupStoreItem)
	app.Post("/groups/:id/store/:itemid/transfer", handlers.GroupStoreItemTransfer)
	app.Get("/groups/:id/transfer", handlers.GroupTransfer)
	app.Get("/groups/:id/transfer/pending", handlers.GroupTransferPending)
	app.Post("/groups/:id/transfer", handlers.GroupTransferPost)
	app.Get("/groups/:id/banner", handlers.GroupBanner)
	app.Get("/groups/:id/icon", handlers.GroupIcon)
	app.Post("/groups/transfers/:id/accept", handlers.GroupTransferAccept)
	app.Post("/groups/transfers/:id/reject", handlers.GroupTransferReject)

	// --- Feed Routes ---
	app.Get("/feed", handlers.FeedPageHandler)
	app.Post("/feed", handlers.PostFeedHandler)
	app.Get("/api/v1/feed", handlers.GetFeedPaginated)
	app.Post("/api/v1/feed", handlers.PostFeed)
	app.Get("/api/v1/feed/comments", handlers.GetFeedCommentsHandler)

	// WebSocket for feed
	app.Get("/ws/feed", websocket.New(handlers.FeedWS))

	// --- Avatar Routes ---
	app.Get("/avatar", handlers.AvatarEditorPage)
	app.Get("/avatar/full-body", handlers.AvatarFullBody)
	app.Get("/avatar/headshot", handlers.AvatarHeadshotPage)
	app.Get("/avatar/:id", handlers.AvatarGet)
	app.Get("/avatar/headshots/:id", handlers.AvatarHeadshotGet)
	app.Get("/avatar/headshot/:id", handlers.AvatarHeadshotGet)
	app.Get("/avatar/shop/:type/:id", handlers.ShopRenderGet)
	app.Get("/avatar/mesh/:id", handlers.AssetRenderGet)
	app.Post("/avatar/color", handlers.AvatarUpdateColorsPost)
	app.Post("/avatar/wear", handlers.AvatarWearItemPost)
	app.Post("/avatar/unwear", handlers.AvatarUnwearItemPost)
	app.Post("/avatar/outfit/save", handlers.AvatarSaveOutfitPost)
	app.Post("/avatar/outfit/wear", handlers.AvatarWearOutfitPost)
	app.Post("/avatar/outfit/delete", handlers.AvatarDeleteOutfitPost)
	app.Post("/avatar/rerender", handlers.AvatarReRenderPost)

	// API v1 avatar routes
	app.Get("/api/v1/avatar/data/:id", handlers.AvatarDataGet)
	app.Get("/api/v1/avatar/inventory", handlers.AvatarInventoryAPI)
	app.Get("/api/v1/avatar/wearing", handlers.AvatarWearingAPI)
	app.Get("/api/v1/avatar/outfits", handlers.AvatarOutfitsAPI)
	app.Get("/api/v1/avatar/outfit/:id", handlers.AvatarOutfitGet)
	app.Get("/api/v1/avatar/:id", handlers.AvatarGet)
	app.Get("/api/v1/avatar/headshots/:id", handlers.AvatarHeadshotGet)
	app.Get("/api/v1/avatar/headshot/:id", handlers.AvatarHeadshotGet)
	app.Get("/api/v1/avatar/shop/:type/:id", handlers.ShopRenderGet)
	app.Get("/api/v1/avatar/mesh/:id", handlers.AssetRenderGet)
	app.Post("/api/v1/avatar/color", handlers.AvatarUpdateColorsPost)
	app.Post("/api/v1/avatar/wear", handlers.AvatarWearItemPost)
	app.Post("/api/v1/avatar/unwear", handlers.AvatarUnwearItemPost)
	app.Post("/api/v1/avatar/outfit/save", handlers.AvatarSaveOutfitPost)
	app.Post("/api/v1/avatar/outfit/wear", handlers.AvatarWearOutfitPost)
	app.Post("/api/v1/avatar/outfit/delete", handlers.AvatarDeleteOutfitPost)
	app.Post("/api/v1/avatar/rerender", handlers.AvatarReRenderPost)

	// --- Inventory ---
	app.Get("/inventory", handlers.InventoryPage)
	app.Get("/api/v1/inventory", handlers.InventoryAPI)

	// --- Shop / Store Routes (luduvo paths) ---
	app.Get("/upgrade", handlers.UpgradePage)
	app.Post("/upgrade/checkout", handlers.UpgradeCheckout)
	app.Post("/store/checkout", handlers.UpgradeCheckout)
	app.Get("/shop", handlers.ShopGet)
	app.Get("/shop/categories", handlers.ShopCategories)
	app.Get("/store", handlers.StorePage)
	app.Get("/store/avatar-rules", handlers.StoreAvatarRules)
	app.Get("/store/:id", handlers.StoreItemDetail)
	app.Post("/store/:id/buy", handlers.StoreBuy)
	app.Get("/store/:id/buy-requests", handlers.StoreBuyRequests)
	app.Post("/store/:id/buy-requests/:brid/cancel", handlers.StoreBuyRequestCancel)
	app.Post("/store/:id/buy-requests/:brid/fill", handlers.StoreBuyRequestFill)
	app.Get("/store/:id/trades/:tradeid", handlers.StoreTradeDetail)
	app.Get("/store/:id/rating", handlers.StoreRating)
	app.Post("/store/:id/rate", handlers.StoreRate)
	app.Get("/store/:id/resells", handlers.StoreResells)
	app.Post("/store/:id/resells/:rid/buy", handlers.StoreResellBuy)
	app.Post("/store/:id/resells/:rid/cancel", handlers.StoreResellCancel)
	app.Post("/store/:id/resells/:rid/gift", handlers.StoreResellGift)
	app.Get("/store/:id/gift", handlers.StoreGift)

	// API v1 shop
	app.Get("/api/v1/shop/items", handlers.ShopItemsAPI)
	app.Get("/api/v1/shop/items/:id", handlers.ShopItemDetailAPI)

	// --- Messages / Conversations ---
	app.Get("/messages", handlers.MessagesPage)
	app.Get("/conversations", handlers.ConversationsList)
	app.Get("/conversations/:id/members", handlers.ConversationMembers)
	app.Get("/conversations/:id/members/me", handlers.ConversationMembersMe)
	app.Get("/conversations/:id/messages", handlers.ConversationMessages)
	app.Post("/conversations/:id/read", handlers.ConversationRead)
	app.Post("/conversations/:id/typing", handlers.ConversationTyping)
	app.Post("/conversations/:id/title", handlers.ConversationTitle)
	app.Post("/conversations/:id/transfer-admin", handlers.ConversationTransferAdmin)

	// --- Trades ---
	app.Get("/trade", handlers.TradePage)
	app.Get("/trades", handlers.TradesList)
	app.Get("/trades/:id", handlers.TradeDetail)
	app.Post("/trades/:id/accept", handlers.TradeAccept)
	app.Post("/trades/:id/decline", handlers.TradeDecline)

	// --- Settings ---
	app.Get("/settings", handlers.SettingsGet)
	app.Post("/settings/displayname", handlers.SettingsDisplayNamePost)
	app.Post("/settings/username", handlers.SettingsUsernamePost)
	app.Post("/settings/password", handlers.SettingsPasswordPost)
	app.Post("/settings/bio", handlers.SettingsBioPost)
	app.Post("/settings/pronouns", handlers.SettingsPronounsPost)
	app.Post("/settings/socials", handlers.SettingsSocialsPost)
	app.Post("/settings/music", handlers.SettingsMusicPost)
	app.Post("/settings/music/remove", handlers.SettingsMusicRemovePost)
	app.Get("/settings/parental-pin", handlers.SettingsParentalPin)
	app.Post("/me/settings", handlers.SettingsPost)

	// --- Items / Marketplace ---
	app.Get("/items", handlers.ItemsPage)
	app.Get("/items/:id/favorite", handlers.ItemsFavorite)
	app.Get("/items/:id/variants", handlers.ItemsVariants)
	app.Get("/items/:id/wishlist", handlers.ItemsWishlist)
	app.Get("/marketplace", handlers.MarketplacePage)
	app.Get("/developer-store", handlers.DeveloperStorePage)

	// --- Notifications ---
	app.Get("/notifications", handlers.NotificationsPage)

	// --- Admin Routes ---
	app.Get("/admin", handlers.AdminIndex)
	app.Get("/admin/users/:id", handlers.AdminUserViewPage)
	app.Post("/admin/users/:id/scrub/:action", handlers.AdminScrubPost)
	app.Post("/admin/users/:id/modhist/:mid/retract", handlers.AdminModhistRetractPost)
	app.Post("/admin/users/:id/resetavatar", handlers.AdminResetAvatarPost)
	app.Post("/admin/users/:id/outfits/:oid/delete", handlers.AdminOutfitDeletePost)
	app.Post("/admin/assets/:id/:action", handlers.AdminAssetReviewPost)

	// API v1 admin
	app.Get("/api/v1/admin/status", handlers.AdminStatusAPI)
	app.Get("/api/v1/admin/users", handlers.AdminUsersAPI)
	app.Get("/api/v1/admin/users/:id", handlers.AdminUserDetailAPI)
	app.Get("/api/v1/admin/logs", handlers.AdminLogsAPI)
	app.Get("/api/v1/admin/assets", handlers.AdminAssetsAPI)
	app.Post("/api/v1/admin/assets/:id/:action", handlers.AdminAssetReviewPost)
	app.Post("/api/v1/admin/users/:id/scrub/:action", handlers.AdminScrubPost)
	app.Post("/api/v1/admin/users/:id/modhist/:mid/retract", handlers.AdminModhistRetractPost)
	app.Post("/api/v1/admin/users/:id/resetavatar", handlers.AdminResetAvatarPost)
	app.Post("/api/v1/admin/users/:id/outfits/:oid/delete", handlers.AdminOutfitDeletePost)

	// --- Asset / Upload Routes ---
	app.Get("/api/v1/assets/:id/file", handlers.AssetFileGet)
	app.Get("/api/v1/assets/:id/render", handlers.AssetRenderGet)
	app.Get("/assets", handlers.AssetsPage)
	app.Get("/assets/:id", handlers.AssetFileGet)
	app.Get("/assets/:id/file", handlers.AssetFileGet)
	app.Get("/assets/:id/render", handlers.AssetRenderGet)
	app.Get("/asset/:id", handlers.AssetFileGet)
	app.Get("/asset/:id/file", handlers.AssetFileGet)
	app.Post("/assets", handlers.AssetUploadPost)
	app.Post("/create/assets", handlers.AssetUploadPost)
	app.Get("/create/assets", handlers.AssetsPage)

	// --- Static Renders ---
	app.Get("/static/renders/avatars/full/:id", handlers.AvatarGet)
	app.Get("/static/renders/avatars/headshots/:id", handlers.AvatarHeadshotGet)
	app.Get("/static/renders/outfits/:id", handlers.AvatarOutfitGet)
	app.Get("/static/renders/meshes/:id", handlers.AssetRenderGet)
	app.Get("/static/renders/shop/:type/:id", handlers.ShopRenderGet)

	// --- Music ---
	app.Get("/api/v1/music/search", handlers.MusicSearchGet)
	app.Get("/api/v1/music/track/:id", handlers.MusicTrackGet)
	app.Post("/api/v1/music/update", handlers.SettingsMusicPost)

	// --- LiveTimer ---
	app.Get("/livetimer", handlers.LiveTimerGet)
	app.Get("/api/v1/livetimer", handlers.LiveTimerAPIGet)

	// --- Various Pages (stubs) ---
	app.Get("/announcements", handlers.AnnouncementsPage)
	app.Get("/events", handlers.EventsStream)
	app.Get("/featured/events", handlers.FeaturedEvents)
	app.Get("/featured/events/:id/click", handlers.FeaturedEventClick)
	app.Get("/leaderboard/rap", handlers.LeaderboardRAP)
	app.Get("/transactions", handlers.TransactionsPage)
	app.Get("/tickets", handlers.TicketsPage)
	app.Get("/tickets/:id/attachments", handlers.TicketAttachments)
	app.Get("/ticket-categories", handlers.TicketCategories)
	app.Get("/appeals", handlers.AppealsPage)
	app.Get("/appeals/:id/attachments", handlers.AppealAttachments)
	app.Get("/reports", handlers.ReportsPage)
	app.Get("/reports/:id/attachments", handlers.ReportAttachments)
	app.Post("/reports", handlers.CreateReport)
	app.Post("/reports/:id/attachments", handlers.ReportAttachmentUpload)
	app.Get("/reports/:id/attachments/:attachmentID", handlers.ReportAttachmentDownload)
	app.Get("/bans", handlers.BansPage)
	app.Get("/block", handlers.BlockPage)
	app.Get("/privacy", handlers.LegalPage)
	app.Get("/terms", handlers.LegalPage)
	app.Get("/community-guidelines", handlers.LegalPage)
	app.Get("/creator-guidelines", handlers.LegalPage)
	app.Get("/safety", handlers.SafetyPage)
	app.Get("/guideline-sections", handlers.GuidelineSections)
	app.Get("/emojis/whitelist", handlers.EmojiWhitelist)
	app.Get("/promocodes", handlers.PromocodesPage)
	app.Get("/site/banners", handlers.SiteBanners)
	app.Get("/site/banners/:id/click", handlers.SiteBannerClick)
	app.Get("/site/promos", handlers.SitePromos)
	app.Get("/cdn/resolve", handlers.CDNResolve)
	app.Post("/cdn/resolve", handlers.CDNResolve)
	app.Get("/legal/current", handlers.LegalCurrent)
	app.Get("/enforcement/evidence/:id", handlers.EnforcementEvidence)
	app.Get("/gifts/:id/accept", handlers.GiftAccept)
	app.Get("/gifts/:id/reject", handlers.GiftReject)
	app.Post("/me/buy-requests/:id/cancel", handlers.BuyRequestCancel)

	// ============================================================
	// VERTIGO ORIGIN ROUTES (keep for backwards compat)
	// ============================================================

	// Profile redirects
	app.Get("/profile", handlers.ProfileRedirect)
	app.Get("/user/:id", handlers.ProfileGet)
	app.Get("/u/:id", handlers.ProfileGet)

	// Friends old paths
	app.Get("/friends", handlers.FriendsGet)
	app.Get("/friends/:id", handlers.FriendsGet)
	app.Post("/friends/send/:id", handlers.FriendSendPost)
	app.Post("/friends/accept/:id", handlers.FriendAcceptPost)
	app.Post("/friends/decline/:id", handlers.FriendDeclinePost)
	app.Post("/friends/cancel/:id", handlers.FriendCancelPost)
	app.Post("/friends/remove/:id", handlers.FriendRemovePost)

	// Settings old paths
	app.Post("/settings/displayname", handlers.SettingsDisplayNamePost)
	app.Post("/settings/username", handlers.SettingsUsernamePost)
	app.Post("/settings/password", handlers.SettingsPasswordPost)
	app.Post("/settings/bio", handlers.SettingsBioPost)
	app.Post("/settings/pronouns", handlers.SettingsPronounsPost)
	app.Post("/settings/socials", handlers.SettingsSocialsPost)
	app.Post("/settings/music", handlers.SettingsMusicPost)
	app.Post("/settings/music/remove", handlers.SettingsMusicRemovePost)

	// Shop old path
	app.Get("/api/v1/shop/items", handlers.ShopItemsAPI)
	app.Get("/api/v1/shop/items/:id", handlers.ShopItemDetailAPI)

	// ============================================================
	// STATIC FILES
	// ============================================================

	app.Get("/static/catalog/:id", handlers.CatalogThumbnail)

	app.Get("/assets*", static.New("./assets", static.Config{
		NotFoundHandler: func(c fiber.Ctx) error {
			return c.Next()
		},
	}))

	app.Get("/static*", static.New("./static", static.Config{
		NotFoundHandler: func(c fiber.Ctx) error {
			return c.Next()
		},
	}))

	app.Get("/*", static.New("./public", static.Config{
		NotFoundHandler: func(c fiber.Ctx) error {
			return c.Next()
		},
	}))

	app.Use(handlers.NotFoundHandler)
}
