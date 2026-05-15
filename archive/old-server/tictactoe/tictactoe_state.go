package tictactoe

import (
	"errors"
	"log"
)

type GameStatus string

const (
	StatusWaiting GameStatus = "waiting"
	StatusActive  GameStatus = "active"
	StatusEnded   GameStatus = "ended"
)

type TictactoeGameState struct {
	Board  [3][3]string
	Turn   string
	Winner string
	Status GameStatus
}

// NewGameState membuat state awal game
func NewGameState(firstTurn string) *TictactoeGameState {
	return &TictactoeGameState{
		Board:  [3][3]string{},
		Turn:   firstTurn,
		Status: StatusWaiting,
	}
}

// ApplyMove adalah SATU-SATUNYA pintu mutasi state
func (gs *TictactoeGameState) ApplyMove(player string, row, col int) error {
	if gs.Status != StatusActive {
		return errors.New("game is not active")
	}

	if gs.Turn != player {
		log.Println(gs.Turn, "gs.Turn")
		log.Println(player, "player")
		return errors.New("not your turn")
	}

	if row < 0 || row > 2 || col < 0 || col > 2 {
		return errors.New("invalid position")
	}

	if gs.Board[row][col] != "" {
		return errors.New("cell already occupied")
	}

	gs.Board[row][col] = player

	if gs.checkWinner(player) {
		gs.Winner = player
		gs.Status = StatusEnded
		return nil
	}

	if gs.isBoardFull() {
		gs.Winner = "Draw"
		gs.Status = StatusEnded
		return nil
	}

	gs.switchTurn()
	return nil
}

// Reset mengembalikan game ke state awal
func (gs *TictactoeGameState) Reset(firstTurn string) {
	gs.Board = [3][3]string{}
	gs.Turn = firstTurn
	gs.Winner = ""
	gs.Status = StatusActive
}

// ===== INTERNAL PURE LOGIC =====

func (gs *TictactoeGameState) switchTurn() {
	if gs.Turn == "X" {
		gs.Turn = "O"
	} else {
		gs.Turn = "X"
	}
}

func (gs *TictactoeGameState) checkWinner(player string) bool {
	board := gs.Board

	for i := 0; i < 3; i++ {
		if board[i][0] == player && board[i][1] == player && board[i][2] == player {
			return true
		}
		if board[0][i] == player && board[1][i] == player && board[2][i] == player {
			return true
		}
	}

	if board[0][0] == player && board[1][1] == player && board[2][2] == player {
		return true
	}

	if board[0][2] == player && board[1][1] == player && board[2][0] == player {
		return true
	}

	return false
}

func (gs *TictactoeGameState) isBoardFull() bool {
	for _, row := range gs.Board {
		for _, cell := range row {
			if cell == "" {
				return false
			}
		}
	}
	return true
}
