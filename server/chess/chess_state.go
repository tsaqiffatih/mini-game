package chess

import (
	"fmt"

	notnil "github.com/notnil/chess"
)

type GameResult struct {
	Status string // ongoing, checkmate, draw
	Winner string // white, black, draw
}

type ChessGameState struct {
	game     *notnil.Game
	isActive bool
	winner   string
}

type ChessMovePayload struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Promotion string `json:"promotion,omitempty"`
}

// ==================
// Constructor
// ==================

func NewChessGameState() *ChessGameState {
	g := notnil.NewGame(notnil.UseNotation(notnil.UCINotation{}))
	return &ChessGameState{
		game:     g,
		isActive: true,
	}
}

func NewChessGameStateFromFEN(fen string) (*ChessGameState, error) {
	pos, err := notnil.FEN(fen)
	if err != nil {
		return nil, fmt.Errorf("invalid fen: %w", err)
	}
	g := notnil.NewGame(pos, notnil.UseNotation(notnil.UCINotation{}))
	return &ChessGameState{
		game:     g,
		isActive: true,
	}, nil
}

// ==================
// Public Getters
// ==================

func (cs *ChessGameState) CurrentTurn() string {
	if cs.game.Position().Turn() == notnil.White {
		return "white"
	}
	return "black"
}

func (cs *ChessGameState) FEN() string {
	return cs.game.FEN()
}

func (cs *ChessGameState) IsActive() bool {
	return cs.isActive
}

func (cs *ChessGameState) Winner() string {
	return cs.winner
}

// ==================
// CORE — TicTacToe-style
// ==================

func (cs *ChessGameState) UpdateState(
	playerMark string,
	from string,
	to string,
	promo string,
) (*GameResult, error) {

	// 1️⃣ game masih aktif?
	if !cs.isActive {
		return nil, fmt.Errorf("game not active")
	}

	// 2️⃣ validasi giliran
	if cs.CurrentTurn() != playerMark {
		return nil, fmt.Errorf("not your turn")
	}

	// 3️⃣ apply move + validasi rule
	if err := cs.applyMove(from, to, promo); err != nil {
		return nil, err
	}

	// 4️⃣ cek akhir game
	if res := cs.checkGameEnd(); res != nil {
		cs.isActive = false
		cs.winner = res.Winner
		return res, nil
	}

	// 5️⃣ game lanjut
	return &GameResult{
		Status: "ongoing",
		Winner: "",
	}, nil
}

// ==================
// Internal helpers
// ==================

func (cs *ChessGameState) applyMove(from, to, promo string) error {
	base := from + to

	// cek apakah move ini butuh promotion
	needsPromotion := false
	for _, mv := range cs.game.ValidMoves() {
		s := mv.String() // UCI
		if len(s) == 5 && s[:4] == base {
			needsPromotion = true
			break
		}
	}

	if needsPromotion && promo == "" {
		return fmt.Errorf("promotion required")
	}

	uci := base
	if promo != "" {
		uci = base + promo // q r b n
	}

	if err := cs.game.MoveStr(uci); err != nil {
		return fmt.Errorf("illegal move: %w", err)
	}

	return nil
}

func (cs *ChessGameState) checkGameEnd() *GameResult {
	pos := cs.game.Position()

	switch pos.Status() {
	case notnil.Checkmate:
		// Turn() adalah pihak yang TIDAK bisa bergerak
		if pos.Turn() == notnil.White {
			return &GameResult{Status: "checkmate", Winner: "black"}
		}
		return &GameResult{Status: "checkmate", Winner: "white"}

	case notnil.Stalemate:
		return &GameResult{Status: "draw", Winner: "draw"}

	case notnil.DrawOffer:
		return &GameResult{Status: "draw", Winner: "draw"}
	}

	return nil
}
