package chess

import (
	"errors"
	"fmt"
	"strings"
	"time"

	notnil "github.com/notnil/chess"
)

const SchemaVersion = 2

type GameResult struct {
	Status string // ongoing, check, checkmate, stalemate, draw
	Winner string // white, black, draw
	Reason string // checkmate, stalemate, draw, insufficient_material, etc.
}

type Piece struct {
	Type  string `json:"type"`
	Color string `json:"color"`
}

type MoveActor struct {
	PlayerID string `json:"player_id"`
	Color    string `json:"color"`
	IsAI     bool   `json:"is_ai"`
}

type MoveFlags struct {
	Capture         bool `json:"capture"`
	Castle          bool `json:"castle"`
	KingsideCastle  bool `json:"kingside_castle"`
	QueensideCastle bool `json:"queenside_castle"`
	EnPassant       bool `json:"en_passant"`
	Promotion       bool `json:"promotion"`
	Check           bool `json:"check"`
	Checkmate       bool `json:"checkmate"`
	Stalemate       bool `json:"stalemate"`
	Draw            bool `json:"draw"`
}

type CheckState struct {
	IsCheck    bool   `json:"is_check"`
	Color      string `json:"color,omitempty"`
	KingSquare string `json:"king_square,omitempty"`
}

type RookMove struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Promotion struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type MoveAnimation struct {
	From           string `json:"from"`
	To             string `json:"to"`
	CapturedSquare string `json:"captured_square,omitempty"`
}

type MoveMetadata struct {
	ID                     string        `json:"id"`
	Ply                    int           `json:"ply"`
	MoveNumber             int           `json:"move_number"`
	Actor                  MoveActor     `json:"actor"`
	From                   string        `json:"from"`
	To                     string        `json:"to"`
	UCI                    string        `json:"uci"`
	SAN                    string        `json:"san"`
	Piece                  Piece         `json:"piece"`
	Captured               *Piece        `json:"captured,omitempty"`
	Promotion              *Promotion    `json:"promotion,omitempty"`
	Flags                  MoveFlags     `json:"flags"`
	RookMove               *RookMove     `json:"rook_move,omitempty"`
	EnPassantCaptureSquare string        `json:"en_passant_capture_square,omitempty"`
	Check                  CheckState    `json:"check"`
	Sound                  string        `json:"sound"`
	Animation              MoveAnimation `json:"animation"`
	CreatedAt              time.Time     `json:"created_at"`
}

type MoveResult struct {
	GameResult *GameResult
	Move       MoveMetadata
}

type HistoryEntry struct {
	Ply       int
	FENBefore string
	FENAfter  string
	Move      MoveMetadata
	UCI       string
	SAN       string
}

type CapturedPieces struct {
	White []Piece `json:"white"`
	Black []Piece `json:"black"`
}

type ChessGameState struct {
	game       *notnil.Game
	isActive   bool
	winner     string
	result     string
	pgnMoves   []string
	history    []HistoryEntry
	lastMove   *MoveMetadata
	captured   CapturedPieces
	status     string
	checkState CheckState
}

func NewChessGameState() *ChessGameState {
	g := notnil.NewGame(notnil.UseNotation(notnil.UCINotation{}))
	return &ChessGameState{
		game:     g,
		isActive: true,
		pgnMoves: []string{},
		status:   "active",
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
		pgnMoves: []string{},
		status:   "active",
	}, nil
}

func (cs *ChessGameState) CurrentTurn() string {
	return colorName(cs.game.Position().Turn())
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

func (cs *ChessGameState) Result() string {
	return cs.result
}

func (cs *ChessGameState) Status() string {
	if cs.status == "" {
		return "active"
	}
	return cs.status
}

func (cs *ChessGameState) PGNMoves() []string {
	return append([]string(nil), cs.pgnMoves...)
}

func (cs *ChessGameState) LastMove() *MoveMetadata {
	if cs.lastMove == nil {
		return nil
	}
	move := *cs.lastMove
	return &move
}

func (cs *ChessGameState) CheckState() CheckState {
	return cs.checkState
}

func (cs *ChessGameState) CapturedPieces() CapturedPieces {
	return CapturedPieces{
		White: append([]Piece(nil), cs.captured.White...),
		Black: append([]Piece(nil), cs.captured.Black...),
	}
}

func (cs *ChessGameState) LegalMoves() map[string][]string {
	legalMoves := map[string][]string{}
	for _, move := range cs.game.ValidMoves() {
		from := move.S1().String()
		legalMoves[from] = append(legalMoves[from], move.S2().String())
	}
	return legalMoves
}

func (cs *ChessGameState) Ply() int {
	return len(cs.history)
}

func (cs *ChessGameState) FullMoveNumber() int {
	fields := strings.Fields(cs.FEN())
	if len(fields) >= 6 {
		var moveNumber int
		if _, err := fmt.Sscanf(fields[5], "%d", &moveNumber); err == nil {
			return moveNumber
		}
	}
	return 1
}

func (cs *ChessGameState) CanUndoAI() bool {
	return len(cs.history) > 0
}

func (cs *ChessGameState) LastUndoablePly() int {
	return len(cs.history)
}

func (cs *ChessGameState) UpdateState(
	playerID string,
	playerMark string,
	isAI bool,
	from string,
	to string,
	promo string,
) (*MoveResult, error) {
	if !cs.isActive {
		return nil, fmt.Errorf("game not active")
	}
	if cs.CurrentTurn() != playerMark {
		return nil, fmt.Errorf("not your turn")
	}

	fenBefore := cs.FEN()
	prePosition := cs.game.Position()
	move, err := cs.findLegalMove(from, to, promo)
	if err != nil {
		return nil, err
	}

	metadata := cs.buildMoveMetadata(prePosition, move, playerID, playerMark, isAI)
	if err := cs.game.Move(move); err != nil {
		return nil, fmt.Errorf("illegal move: %w", err)
	}

	gameResult := cs.updateStatusAfterMove()
	metadata.Flags.Checkmate = gameResult.Status == "checkmate"
	metadata.Flags.Stalemate = gameResult.Status == "stalemate"
	metadata.Flags.Draw = gameResult.Winner == "draw"
	metadata.Check = cs.checkState
	metadata.Sound = moveSound(metadata.Flags, gameResult)
	if metadata.Flags.EnPassant {
		metadata.Animation.CapturedSquare = metadata.EnPassantCaptureSquare
	}

	cs.lastMove = &metadata
	cs.pgnMoves = append(cs.pgnMoves, metadata.SAN)
	cs.appendCapturedPiece(metadata.Captured)
	cs.history = append(cs.history, HistoryEntry{
		Ply:       metadata.Ply,
		FENBefore: fenBefore,
		FENAfter:  cs.FEN(),
		Move:      metadata,
		UCI:       metadata.UCI,
		SAN:       metadata.SAN,
	})

	return &MoveResult{
		GameResult: gameResult,
		Move:       metadata,
	}, nil
}

func (cs *ChessGameState) RollbackLastAITurn() error {
	if len(cs.history) == 0 {
		return errors.New("no moves to undo")
	}

	targetPly := len(cs.history) - 1
	if cs.history[len(cs.history)-1].Move.Actor.IsAI && targetPly > 0 {
		targetPly--
	}
	return cs.RollbackToPly(targetPly)
}

func (cs *ChessGameState) RollbackToPly(targetPly int) error {
	if targetPly < 0 || targetPly > len(cs.history) {
		return errors.New("invalid rollback ply")
	}

	targetFEN := notnil.StartingPosition().String()
	if targetPly > 0 {
		targetFEN = cs.history[targetPly-1].FENAfter
	}

	pos, err := notnil.FEN(targetFEN)
	if err != nil {
		return fmt.Errorf("rollback fen invalid: %w", err)
	}

	cs.game = notnil.NewGame(pos, notnil.UseNotation(notnil.UCINotation{}))
	cs.history = append([]HistoryEntry(nil), cs.history[:targetPly]...)
	cs.pgnMoves = make([]string, 0, len(cs.history))
	cs.captured = CapturedPieces{}
	for _, entry := range cs.history {
		cs.pgnMoves = append(cs.pgnMoves, entry.SAN)
		cs.appendCapturedPiece(entry.Move.Captured)
	}

	cs.isActive = true
	cs.winner = ""
	cs.result = ""
	cs.status = "active"
	cs.checkState = CheckState{}
	cs.lastMove = nil
	if len(cs.history) > 0 {
		last := cs.history[len(cs.history)-1].Move
		cs.lastMove = &last
		cs.checkState = last.Check
		if last.Flags.Check {
			cs.status = "check"
		}
	}
	return nil
}

func (cs *ChessGameState) findLegalMove(from, to, promo string) (*notnil.Move, error) {
	base := from + to
	needsPromotion := false
	for _, move := range cs.game.ValidMoves() {
		uci := move.String()
		if len(uci) >= 4 && uci[:4] == base {
			if len(uci) == 5 {
				needsPromotion = true
			}
			if promo == "" || strings.EqualFold(uci, base+promo) {
				if needsPromotion && promo == "" {
					continue
				}
				return move, nil
			}
		}
	}
	if needsPromotion && promo == "" {
		return nil, fmt.Errorf("promotion required")
	}
	return nil, fmt.Errorf("illegal move")
}

func (cs *ChessGameState) buildMoveMetadata(prePosition *notnil.Position, move *notnil.Move, playerID string, playerMark string, isAI bool) MoveMetadata {
	movedPiece := prePosition.Board().Piece(move.S1())
	capturedPiece := prePosition.Board().Piece(move.S2())
	capturedSquare := move.S2().String()
	if move.HasTag(notnil.EnPassant) {
		if movedPiece.Color() == notnil.White {
			capturedSquare = notnil.NewSquare(move.S2().File(), move.S2().Rank()-1).String()
		} else {
			capturedSquare = notnil.NewSquare(move.S2().File(), move.S2().Rank()+1).String()
		}
		capturedPiece = notnil.NewPiece(notnil.Pawn, movedPiece.Color().Other())
	}

	var captured *Piece
	if capturedPiece != notnil.NoPiece {
		piece := pieceDTO(capturedPiece)
		captured = &piece
	}

	flags := MoveFlags{
		Capture:         move.HasTag(notnil.Capture) || move.HasTag(notnil.EnPassant),
		KingsideCastle:  move.HasTag(notnil.KingSideCastle),
		QueensideCastle: move.HasTag(notnil.QueenSideCastle),
		EnPassant:       move.HasTag(notnil.EnPassant),
		Promotion:       move.Promo() != notnil.NoPieceType,
		Check:           move.HasTag(notnil.Check),
	}
	flags.Castle = flags.KingsideCastle || flags.QueensideCastle

	var promotion *Promotion
	if flags.Promotion {
		promotion = &Promotion{From: "pawn", To: pieceTypeName(move.Promo())}
	}

	metadata := MoveMetadata{
		ID:         fmt.Sprintf("%d", len(cs.history)+1),
		Ply:        len(cs.history) + 1,
		MoveNumber: (len(cs.history) / 2) + 1,
		Actor: MoveActor{
			PlayerID: playerID,
			Color:    playerMark,
			IsAI:     isAI,
		},
		From:      move.S1().String(),
		To:        move.S2().String(),
		UCI:       move.String(),
		SAN:       notnil.AlgebraicNotation{}.Encode(prePosition, move),
		Piece:     pieceDTO(movedPiece),
		Captured:  captured,
		Promotion: promotion,
		Flags:     flags,
		RookMove:  rookMoveDTO(movedPiece.Color(), flags),
		Animation: MoveAnimation{
			From:           move.S1().String(),
			To:             move.S2().String(),
			CapturedSquare: capturedSquareIfAny(flags, capturedSquare),
		},
		CreatedAt: time.Now().UTC(),
	}
	if flags.EnPassant {
		metadata.EnPassantCaptureSquare = capturedSquare
	}
	return metadata
}

func (cs *ChessGameState) updateStatusAfterMove() *GameResult {
	pos := cs.game.Position()
	cs.status = "active"
	cs.result = ""
	cs.winner = ""
	cs.checkState = CheckState{}

	switch pos.Status() {
	case notnil.Checkmate:
		cs.isActive = false
		winner := colorName(pos.Turn().Other())
		cs.winner = winner
		cs.result = "checkmate"
		cs.status = "checkmate"
		cs.checkState = CheckState{IsCheck: true, Color: colorName(pos.Turn()), KingSquare: kingSquare(pos, pos.Turn())}
		return &GameResult{Status: "checkmate", Winner: winner, Reason: "checkmate"}
	case notnil.Stalemate:
		cs.isActive = false
		cs.winner = "draw"
		cs.result = "stalemate"
		cs.status = "stalemate"
		return &GameResult{Status: "stalemate", Winner: "draw", Reason: "stalemate"}
	case notnil.DrawOffer:
		cs.isActive = false
		cs.winner = "draw"
		cs.result = "draw"
		cs.status = "draw"
		return &GameResult{Status: "draw", Winner: "draw", Reason: "draw"}
	}

	last := cs.game.Moves()
	if len(last) > 0 && last[len(last)-1].HasTag(notnil.Check) {
		cs.status = "check"
		cs.checkState = CheckState{IsCheck: true, Color: colorName(pos.Turn()), KingSquare: kingSquare(pos, pos.Turn())}
		return &GameResult{Status: "check", Winner: "", Reason: "check"}
	}

	return &GameResult{Status: "ongoing", Winner: "", Reason: ""}
}

func (cs *ChessGameState) appendCapturedPiece(piece *Piece) {
	if piece == nil {
		return
	}
	switch piece.Color {
	case "white":
		cs.captured.White = append(cs.captured.White, *piece)
	case "black":
		cs.captured.Black = append(cs.captured.Black, *piece)
	}
}

func pieceDTO(piece notnil.Piece) Piece {
	return Piece{Type: pieceTypeName(piece.Type()), Color: colorName(piece.Color())}
}

func pieceTypeName(pieceType notnil.PieceType) string {
	switch pieceType {
	case notnil.King:
		return "king"
	case notnil.Queen:
		return "queen"
	case notnil.Rook:
		return "rook"
	case notnil.Bishop:
		return "bishop"
	case notnil.Knight:
		return "knight"
	case notnil.Pawn:
		return "pawn"
	default:
		return ""
	}
}

func colorName(color notnil.Color) string {
	switch color {
	case notnil.White:
		return "white"
	case notnil.Black:
		return "black"
	default:
		return ""
	}
}

func kingSquare(pos *notnil.Position, color notnil.Color) string {
	for square, piece := range pos.Board().SquareMap() {
		if piece.Type() == notnil.King && piece.Color() == color {
			return square.String()
		}
	}
	return ""
}

func rookMoveDTO(color notnil.Color, flags MoveFlags) *RookMove {
	if !flags.Castle {
		return nil
	}
	if color == notnil.White && flags.KingsideCastle {
		return &RookMove{From: "h1", To: "f1"}
	}
	if color == notnil.White && flags.QueensideCastle {
		return &RookMove{From: "a1", To: "d1"}
	}
	if color == notnil.Black && flags.KingsideCastle {
		return &RookMove{From: "h8", To: "f8"}
	}
	if color == notnil.Black && flags.QueensideCastle {
		return &RookMove{From: "a8", To: "d8"}
	}
	return nil
}

func capturedSquareIfAny(flags MoveFlags, square string) string {
	if flags.Capture || flags.EnPassant {
		return square
	}
	return ""
}

func moveSound(flags MoveFlags, result *GameResult) string {
	if result != nil {
		switch result.Status {
		case "checkmate":
			return "checkmate"
		case "stalemate", "draw":
			return "draw"
		case "check":
			return "check"
		}
	}
	if flags.Promotion && flags.Capture {
		return "promotion_capture"
	}
	if flags.Promotion {
		return "promotion"
	}
	if flags.Castle {
		return "castle"
	}
	if flags.Capture {
		return "capture"
	}
	return "move"
}
