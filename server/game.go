package main

import (
	"math/rand"
)

type GameStatus string

const (
	StatusWaiting  GameStatus = "waiting"  // room created, waiting for player 2
	StatusPlaying  GameStatus = "playing"
	StatusFinished GameStatus = "finished"
)

type GameState struct {
	Rows          [4]int     `json:"rows"`
	TotalSticks   int        `json:"totalSticks"`
	CurrentPlayer int        `json:"currentPlayer"` // 0 or 1
	Status        GameStatus `json:"status"`
	Winner        int        `json:"winner"` // -1 if none
	Loser         int        `json:"loser"`  // -1 if none
	Players       [2]string  `json:"players"`
	RoomCode      string     `json:"roomCode"`
}

type Move struct {
	RowIndex  int    `json:"rowIndex"`
	Direction string `json:"direction"` // "left" | "right"
	Count     int    `json:"count"`
}

func newGame(roomCode string, p1Name string) GameState {
	return GameState{
		Rows:          [4]int{1, 3, 5, 7},
		TotalSticks:   16,
		CurrentPlayer: 0,
		Status:        StatusWaiting,
		Winner:        -1,
		Loser:         -1,
		Players:       [2]string{p1Name, ""},
		RoomCode:      roomCode,
	}
}

func (g *GameState) startPlaying(firstPlayer int) {
	g.Status = StatusPlaying
	if firstPlayer == -1 {
		g.CurrentPlayer = rand.Intn(2)
	} else {
		g.CurrentPlayer = firstPlayer
	}
}

func (g *GameState) applyMove(m Move) bool {
	if g.Status != StatusPlaying {
		return false
	}
	row := g.Rows[m.RowIndex]
	if row == 0 || m.Count < 1 || m.Count > row {
		return false
	}

	g.Rows[m.RowIndex] -= m.Count
	g.TotalSticks -= m.Count

	if g.TotalSticks == 0 {
		g.Loser = g.CurrentPlayer
		g.Winner = 1 - g.CurrentPlayer
		g.Status = StatusFinished
	} else {
		g.CurrentPlayer = 1 - g.CurrentPlayer
	}
	return true
}

func (g *GameState) reset(firstPlayer int) {
	g.Rows = [4]int{1, 3, 5, 7}
	g.TotalSticks = 16
	g.Winner = -1
	g.Loser = -1
	g.Status = StatusPlaying
	if firstPlayer == -1 {
		g.CurrentPlayer = rand.Intn(2)
	} else {
		g.CurrentPlayer = firstPlayer
	}
}
