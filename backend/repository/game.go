package repository

import (
	"database/sql"
	"errors"

	"buildoverse/backend/models"
)

type GameRepository struct {
	db *sql.DB
}

func NewGameRepository(db *sql.DB) *GameRepository {
	return &GameRepository{db: db}
}

func (r *GameRepository) GetPopularGames(limit int) ([]*models.Game, error) {
	if r.db == nil {
		return nil, nil
	}
	query := "SELECT id, name, description, creatorid, genre, visits, playing, server, private, locked, createdat, editedat, tags FROM games WHERE private = 0 AND locked = 0 ORDER BY visits DESC, id DESC LIMIT ?"
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []*models.Game
	for rows.Next() {
		var g models.Game
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatorID, &g.Genre, &g.Visits, &g.Playing, &g.Server, &g.Private, &g.Locked, &g.CreatedAt, &g.EditedAt, &g.Tags); err != nil {
			return nil, err
		}
		games = append(games, &g)
	}
	return games, nil
}

func (r *GameRepository) GetGameByID(id int) (*models.Game, error) {
	if r.db == nil {
		return nil, errors.New("game database is not configured")
	}

	query := "SELECT id, name, description, creatorid, genre, visits, playing, server, private, locked, createdat, editedat, tags FROM games WHERE id = ? AND private = 0 AND locked = 0"
	var game models.Game
	err := r.db.QueryRow(query, id).Scan(
		&game.ID,
		&game.Name,
		&game.Description,
		&game.CreatorID,
		&game.Genre,
		&game.Visits,
		&game.Playing,
		&game.Server,
		&game.Private,
		&game.Locked,
		&game.CreatedAt,
		&game.EditedAt,
		&game.Tags,
	)
	if err != nil {
		return nil, err
	}
	return &game, nil
}

func (r *GameRepository) GetGamesByCreatorID(creatorID, limit, offset int) ([]*models.Game, int, error) {
	if r.db == nil {
		return nil, 0, errors.New("game database is not configured")
	}
	if creatorID <= 0 {
		return nil, 0, errors.New("invalid creator ID")
	}

	var total int
	if err := r.db.QueryRow(
		"SELECT COUNT(*) FROM games WHERE creatorid = ? AND private = 0 AND locked = 0",
		creatorID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(
		`SELECT id, name, description, creatorid, genre, visits, playing, server,
		        private, locked, createdat, editedat, tags
		 FROM games
		 WHERE creatorid = ? AND private = 0 AND locked = 0
		 ORDER BY editedat DESC, id DESC
		 LIMIT ? OFFSET ?`,
		creatorID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	games := make([]*models.Game, 0)
	for rows.Next() {
		var game models.Game
		if err := rows.Scan(
			&game.ID, &game.Name, &game.Description, &game.CreatorID, &game.Genre,
			&game.Visits, &game.Playing, &game.Server, &game.Private, &game.Locked,
			&game.CreatedAt, &game.EditedAt, &game.Tags,
		); err != nil {
			return nil, 0, err
		}
		games = append(games, &game)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return games, total, nil
}
