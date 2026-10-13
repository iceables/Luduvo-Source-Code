package service

import (
	"database/sql"
	"time"

	"buildoverse/backend/repository"
)

var (
	Auth       *AuthService
	User       *UserService
	Feed       *FeedService
	Game       *GameService
	Music      *MusicService
	Cooldown   *CooldownService
	Friend     *FriendService
	ModHistory *ModHistoryService
	Avatar     *AvatarService
	Shop       *ShopService
	Asset      *AssetService
)

func Init(db *sql.DB) {
	userRepo := repository.NewUserRepository(db)
	feedRepo := repository.NewFeedRepository(db)
	gameRepo := repository.NewGameRepository(db)
	friendRepo := repository.NewFriendRepository(db)
	modHistRepo := repository.NewModHistoryRepository(db)
	avatarRepo := repository.NewAvatarRepository(db)
	shopRepo := repository.NewShopRepository(db)
	assetRepo := repository.NewAssetRepository(db)

	Cooldown = NewCooldownService(5 * time.Second)
	Auth = NewAuthService(userRepo)
	User = NewUserService(userRepo, Cooldown)
	Feed = NewFeedService(feedRepo, userRepo, Cooldown)
	Game = NewGameService(gameRepo)
	Music = NewMusicService()
	Friend = NewFriendService(friendRepo, userRepo, Cooldown)
	ModHistory = NewModHistoryService(modHistRepo, userRepo)
	Avatar = NewAvatarService(avatarRepo, userRepo)
	Shop = NewShopService(shopRepo)
	Asset = NewAssetService(assetRepo)
}