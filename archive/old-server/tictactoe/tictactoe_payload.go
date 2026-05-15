package tictactoe

type TictactoeMovePayload struct {
	RoomID   string `json:"room_id"`
	PlayerID string `json:"player_id"`
	Row      int    `json:"row"`
	Col      int    `json:"col"`
}

type TicTacToeGameResponse struct {
	Board    [3][3]string `json:"board"`
	Turn     string       `json:"turn"`
	Winner   string       `json:"winner"`
	IsActive bool         `json:"is_active"`
}
