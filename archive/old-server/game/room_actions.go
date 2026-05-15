package game

import (
	"errors"

	"github.com/tsaqiffatih/mini-game/tictactoe"
)

type MoveResult struct {
	GameEnded bool
	Result    interface{}
	Err       error
}

var (
	ErrPlayerNotFound   = errors.New("player not found")
	ErrInvalidGameState = errors.New("invalid game state")
)

// === TicTacToe ===
func (r *Room) ApplyTicTacToeMove(playerID string, row, col int) (*tictactoe.TictactoeGameState, bool, error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	player, ok := r.Players[playerID]
	if !ok {
		return nil, false, ErrPlayerNotFound
	}

	gs, ok := r.GameState.Data.(*tictactoe.TictactoeGameState)
	if !ok {
		return nil, false, ErrInvalidGameState
	}

	err := gs.ApplyMove(player.Mark, row, col)
	ended := gs.Status == tictactoe.StatusEnded

	return gs, ended, err
}
