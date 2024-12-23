package tictactoe

import (
	"errors"
	"sync"
)

type TictactoeGameState struct {
	Board    [3][3]string
	Turn     string
	Winner   string
	IsActive bool
	mu       sync.Mutex
	updates  chan Update
}

type TicTacToeGameResponse struct {
	Board    [3][3]string `json:"board"`
	Turn     string       `json:"turn"`
	Winner   string       `json:"winner"`
	IsActive bool         `json:"is_active"`
}

type Update struct {
	PlayerMark string
	Row        int
	Col        int
	Err        chan error
}

type TictactoeMovePayload struct {
	RoomID   string `json:"room_id"`
	PlayerID string `json:"player_id"`
	Row      int    `json:"row"`
	Col      int    `json:"col"`
}

// di inisialisasi saat user CreateRoom
func NewGameState() *TictactoeGameState {
	gs := &TictactoeGameState{
		Board:    [3][3]string{},
		Turn:     "X",
		IsActive: false,
		updates:  make(chan Update),
	}
	go gs.run()
	return gs
}

func (gs *TictactoeGameState) run() {
	for update := range gs.updates {
		gs.mu.Lock()
		if !gs.IsActive {
			update.Err <- errors.New("game is not active")
		} else if gs.Board[update.Row][update.Col] != "" {
			update.Err <- errors.New("cell is already occupied")
		} else if gs.Turn != update.PlayerMark {
			update.Err <- errors.New("not your turn")
		} else {
			gs.Board[update.Row][update.Col] = update.PlayerMark
			if gs.checkWinner(update.PlayerMark) {
				gs.Winner = update.PlayerMark
				gs.IsActive = false
			} else if gs.isBoardFull() {
				gs.Winner = "Draw"
				gs.IsActive = false
			} else {
				gs.switchTurn()
			}
			update.Err <- nil
		}
		gs.mu.Unlock()
	}
}

func (gs *TictactoeGameState) UpdateState(playerMark string, row, col int) error {
	errChan := make(chan error)
	gs.updates <- Update{
		PlayerMark: playerMark,
		Row:        row,
		Col:        col,
		Err:        errChan,
	}
	return <-errChan
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

func (gs *TictactoeGameState) ResetGame() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.Board = [3][3]string{}
	gs.Turn = "X"
	gs.Winner = ""
	gs.IsActive = true
}

func (gs *TictactoeGameState) checkWinner(mark string) bool {
	lines := [][][2]int{
		{{0, 0}, {0, 1}, {0, 2}},
		{{1, 0}, {1, 1}, {1, 2}},
		{{2, 0}, {2, 1}, {2, 2}},
		{{0, 0}, {1, 0}, {2, 0}},
		{{0, 1}, {1, 1}, {2, 1}},
		{{0, 2}, {1, 2}, {2, 2}},
		{{0, 0}, {1, 1}, {2, 2}},
		{{0, 2}, {1, 1}, {2, 0}},
	}

	for _, line := range lines {
		if gs.Board[line[0][0]][line[0][1]] == mark &&
			gs.Board[line[1][0]][line[1][1]] == mark &&
			gs.Board[line[2][0]][line[2][1]] == mark {
			return true
		}
	}
	return false
}

func (gs *TictactoeGameState) switchTurn() {
	if gs.Turn == "X" {
		gs.Turn = "O"
	} else {
		gs.Turn = "X"
	}
}

func (gs *TictactoeGameState) GetState() ([3][3]string, string, string, bool) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.Board, gs.Turn, gs.Winner, gs.IsActive
}
