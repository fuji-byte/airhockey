package models

import (
	"github.com/gorilla/websocket"
)

type GameState struct {
	PuckX      float64 `json:"PuckX"`
	PuckY      float64 `json:"PuckY"`
	PuckSpeedX float64 `json:"PuckSpeedX"`
	PuckSpeedY float64 `json:"PuckSpeedY"`
	Player1X   float64 `json:"Player1X"`
	Player1Y   float64 `json:"Player1Y"`
	Player2X   float64 `json:"Player2X"`
	Player2Y   float64 `json:"Player2Y"`
	Score1     int     `json:"Score1"`
	Score2     int     `json:"Score2"`
	Time       int     `json:"Time"`
}

type Room struct {
	RoomID    int
	Players   []*Client
	GameState *GameState
	// Mutex     sync.Mutex
}

type Score struct { //バトル履歴などをデータベースに保存する
	ID       string `json:"id"`
	ClientID string `json:"client_id"`
	Score    int    `json:"score"`
}

type Client struct { //memory
	ID     string          //ClienID
	Conn   *websocket.Conn `gorm:"-"`
	RoomID int
	Key    string
}

type InputNum struct {
	RoomID int
}

// type OutputNum struct {
// 	UserNum int
// }

type Message struct {
	Type      string    `json:"type"`
	RoomID    int       `json:"roomnum"`
	GameState GameState `json:"gamestate"`
	Message   string    `json:"message"`
}
