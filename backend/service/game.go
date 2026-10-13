package service

import (
	"buildoverse/backend/models"
	"buildoverse/backend/repository"
)

type GameService struct {
	gameRepo *repository.GameRepository
}

func NewGameService(gameRepo *repository.GameRepository) *GameService {
	return &GameService{gameRepo: gameRepo}
}

func (s *GameService) GetPopularGames(limit int) ([]*models.Game, error) {
	return s.gameRepo.GetPopularGames(limit)
}

func (s *GameService) GetGameByID(id int) (*models.Game, error) {
	return s.gameRepo.GetGameByID(id)
}

func (s *GameService) GetGamesByCreatorID(creatorID, limit, offset int) ([]*models.Game, int, error) {
	return s.gameRepo.GetGamesByCreatorID(creatorID, limit, offset)
}
