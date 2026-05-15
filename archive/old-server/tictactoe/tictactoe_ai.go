package tictactoe

import "math"

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
