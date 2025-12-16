package chess

import (
	"fmt"

	notnil "github.com/notnil/chess"
)

type ChessGameState struct {
	Game *notnil.Game
}

func NewChessGameState() *ChessGameState {
	g := notnil.NewGame(notnil.UseNotation(notnil.UCINotation{}))
	return &ChessGameState{Game: g}
}

func NewChessGameStateFromFEN(fenStr string) (*ChessGameState, error) {
	pos, err := notnil.FEN(fenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid fen: %w", err)
	}
	g := notnil.NewGame(pos, notnil.UseNotation(notnil.UCINotation{}))
	return &ChessGameState{Game: g}, nil
}

func (c *ChessGameState) CurrentTurn() string {
	if c.Game.Position().Turn() == notnil.White {
		return "white"
	}
	return "black"
}

func (c *ChessGameState) ApplyMove(from, to string, promo string, player string) error {
	// 1. Validate turn ownership
	// 1) Validate turn ownership
	if (c.CurrentTurn() == "white" && player != "white") ||
		(c.CurrentTurn() == "black" && player != "black") {
		return fmt.Errorf("not player's turn")
	}

	// 2) Build base UCI (without promo)
	base := from + to

	// 3) Check if any valid move requires promotion for this from->to
	needsPromotion := false
	for _, mv := range c.ValidMoves() {
		// promotion moves will be length 5 like "e7e8q"
		if len(mv) == 5 && mv[:4] == base {
			needsPromotion = true
			break
		}
	}

	// 4) If promotion is needed but no promo provided => return explicit error
	if needsPromotion && promo == "" {
		return fmt.Errorf("promotion required but not provided")
	}

	// 5) Build final move string
	uci := base
	if promo != "" {
		// ensure promo is single-letter q/r/b/n (caller responsibility)
		uci = base + promo
	}

	// 6) Try to apply move (MoveStr uses game's notation; MoveStr will return error if illegal)
	if err := c.Game.MoveStr(uci); err != nil {
		return fmt.Errorf("illegal move: %w", err)
	}

	return nil
}

// Return full FEN like chess.js
func (c *ChessGameState) FEN() string {
	return c.Game.FEN()
}

func (c *ChessGameState) ValidMoves() []string {
	moves := c.Game.ValidMoves()
	out := make([]string, 0, len(moves))
	for _, m := range moves {
		out = append(out, m.String()) // ensure UCI
	}
	return out
}
