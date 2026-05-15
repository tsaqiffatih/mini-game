package tictactoe

import (
	"math"
	"math/rand"
)

type Move struct {
	Row int
	Col int
}

func ComputeBestMove(gs *TictactoeGameState, aiMark string) Move {
	bestScore := math.MinInt
	bestMove := Move{Row: -1, Col: -1}

	board := gs.Board
	opponent := opposite(aiMark)

	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if board[r][c] == "" {
				board[r][c] = aiMark
				score := minimax(board, false, aiMark, opponent, 0, math.MinInt, math.MaxInt)
				board[r][c] = ""

				if score > bestScore {
					bestScore = score
					bestMove = Move{Row: r, Col: c}
				}
			}
		}
	}

	return bestMove
}

func ComputeMove(gs *TictactoeGameState, aiMark string, level int) Move {
	availableMoves := availableMoves(gs.Board)
	if len(availableMoves) == 0 {
		return Move{Row: -1, Col: -1}
	}

	if shouldPlayOptimal(level) {
		return ComputeBestMove(gs, aiMark)
	}

	return availableMoves[rand.Intn(len(availableMoves))]
}

// ================= INTERNAL =================

func minimax(
	board [3][3]string,
	isMaximizing bool,
	aiMark, opponent string,
	depth int,
	alpha, beta int,
) int {

	if winner := evaluateWinner(board); winner != "" {
		switch winner {
		case aiMark:
			return 10 - depth
		case opponent:
			return depth - 10
		case "Draw":
			return 0
		}
	}

	if isMaximizing {
		best := math.MinInt
		for r := 0; r < 3; r++ {
			for c := 0; c < 3; c++ {
				if board[r][c] == "" {
					board[r][c] = aiMark
					score := minimax(board, false, aiMark, opponent, depth+1, alpha, beta)
					board[r][c] = ""

					best = max(best, score)
					alpha = max(alpha, best)
					if beta <= alpha {
						return best
					}
				}
			}
		}
		return best
	}

	// minimizing
	best := math.MaxInt
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if board[r][c] == "" {
				board[r][c] = opponent
				score := minimax(board, true, aiMark, opponent, depth+1, alpha, beta)
				board[r][c] = ""

				best = min(best, score)
				beta = min(beta, best)
				if beta <= alpha {
					return best
				}
			}
		}
	}
	return best
}

func evaluateWinner(board [3][3]string) string {
	lines := [8][3][2]int{
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
		a, b, c := line[0], line[1], line[2]
		if board[a[0]][a[1]] != "" &&
			board[a[0]][a[1]] == board[b[0]][b[1]] &&
			board[a[0]][a[1]] == board[c[0]][c[1]] {
			return board[a[0]][a[1]]
		}
	}

	for _, row := range board {
		for _, cell := range row {
			if cell == "" {
				return ""
			}
		}
	}

	return "Draw"
}

func availableMoves(board [3][3]string) []Move {
	moves := []Move{}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if board[r][c] == "" {
				moves = append(moves, Move{Row: r, Col: c})
			}
		}
	}
	return moves
}

func shouldPlayOptimal(level int) bool {
	if level < 1 {
		level = 1
	}
	if level > 10 {
		level = 10
	}

	chance := map[int]int{
		1:  10,
		2:  20,
		3:  35,
		4:  45,
		5:  60,
		6:  75,
		7:  85,
		8:  92,
		9:  100,
		10: 100,
	}[level]

	return rand.Intn(100) < chance
}

func opposite(mark string) string {
	if mark == "X" {
		return "O"
	}
	return "X"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
