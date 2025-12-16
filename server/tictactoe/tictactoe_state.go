package tictactoe

import (
	"errors"
	"log"
	"math"
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

// NewGameState initializes a new Tic Tac Toe game state.
// It sets up the board, turn, and other initial values.
func NewTicTacToeGameState() *TictactoeGameState {
	gs := &TictactoeGameState{
		Board:    [3][3]string{},
		Turn:     "X",
		IsActive: false,
		updates:  make(chan Update),
	}
	go gs.run()
	return gs
}

// run listens for updates to the game state and processes them.
// It handles moves, checks for winners, and updates the game state accordingly.
func (gs *TictactoeGameState) run() {
	for update := range gs.updates {
		gs.mu.Lock()
		if !gs.IsActive {
			log.Println("Game is not active")
			update.Err <- errors.New("game is not active")
		} else if gs.Board[update.Row][update.Col] != "" {
			log.Println("Cell is already occupied")
			update.Err <- errors.New("cell is already occupied")
		} else if gs.Turn != update.PlayerMark {
			log.Println("Not your turn")
			update.Err <- errors.New("not your turn")
		} else {
			gs.Board[update.Row][update.Col] = update.PlayerMark
			if gs.checkWinner(update.PlayerMark) {
				gs.Winner = update.PlayerMark
				gs.IsActive = false
				log.Printf("Player %s wins", update.PlayerMark)
			} else if gs.isBoardFull() {
				gs.Winner = "Draw"
				gs.IsActive = false
				log.Println("Game is a draw")
			} else {
				gs.switchTurn()
			}
			update.Err <- nil
		}
		gs.mu.Unlock()
	}
}

// UpdateState processes a player's move and updates the game state.
// It validates the move and checks for errors.
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

// isBoardFull checks if the game board is completely filled.
// Returns true if no empty cells are left.
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

// ResetGame resets the game state to its initial values.
// It clears the board and sets the turn to "X".
func (gs *TictactoeGameState) ResetGame() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.Board = [3][3]string{}
	gs.Turn = "X"
	gs.Winner = ""
	gs.IsActive = true
}

// checkWinner checks if a player has won the game.
// It evaluates all possible winning lines on the board.
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

// switchTurn switches the turn between players "X" and "O".
func (gs *TictactoeGameState) switchTurn() {
	if gs.Turn == "X" {
		gs.Turn = "O"
	} else {
		gs.Turn = "X"
	}
}

// GetState retrieves the current state of the game.
// It returns the board, turn, winner, and active status.
func (gs *TictactoeGameState) GetState() ([3][3]string, string, string, bool) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.Board, gs.Turn, gs.Winner, gs.IsActive
}

// MakeAIMove makes a move for the AI player.
func (gs *TictactoeGameState) MakeAIMove() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if !gs.IsActive {
		return
	}

	bestScore := math.Inf(-1)
	var bestMove [2]int

	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if gs.Board[row][col] == "" {
				gs.Board[row][col] = gs.Turn
				score := minimax(gs.Board, 0, false, gs.Turn, gs.getOpponent())
				gs.Board[row][col] = ""
				if score > bestScore {
					bestScore = score
					bestMove = [2]int{row, col}
				}
			}
		}
	}

	row, col := bestMove[0], bestMove[1]
	gs.Board[row][col] = gs.Turn
	if gs.checkWinner(gs.Turn) {
		gs.Winner = gs.Turn
		gs.IsActive = false
	} else if gs.isBoardFull() {
		gs.Winner = "Draw"
		gs.IsActive = false
	} else {
		gs.switchTurn()
	}
}

func minimax(board [3][3]string, depth int, isMaximizing bool, aiMark, playerMark string) float64 {
	// Add a depth limit to minimize computation
	const depthLimit = 5
	if depth >= depthLimit {
		return 0 // Neutral score for deeper levels
	}

	winner := checkWinnerStatic(board)
	if winner == aiMark {
		return 10 - float64(depth)
	} else if winner == playerMark {
		return float64(depth) - 10
	} else if isFullStatic(board) {
		return 0
	}

	if isMaximizing {
		best := math.Inf(-1)
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if board[i][j] == "" {
					board[i][j] = aiMark
					score := minimax(board, depth+1, false, aiMark, playerMark)
					board[i][j] = ""
					best = math.Max(best, score)
				}
			}
		}
		return best
	} else {
		best := math.Inf(1)
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if board[i][j] == "" {
					board[i][j] = playerMark
					score := minimax(board, depth+1, true, aiMark, playerMark)
					board[i][j] = ""
					best = math.Min(best, score)
				}
			}
		}
		return best
	}
}

func (gs *TictactoeGameState) getOpponent() string {
	if gs.Turn == "X" {
		return "O"
	}
	return "X"
}

func checkWinnerStatic(board [3][3]string) string {
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
		if board[line[0][0]][line[0][1]] != "" &&
			board[line[0][0]][line[0][1]] == board[line[1][0]][line[1][1]] &&
			board[line[1][0]][line[1][1]] == board[line[2][0]][line[2][1]] {
			return board[line[0][0]][line[0][1]]
		}
	}
	return ""
}

func isFullStatic(board [3][3]string) bool {
	for _, row := range board {
		for _, cell := range row {
			if cell == "" {
				return false
			}
		}
	}
	return true
}
