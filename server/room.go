package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Message types client → server
const (
	MsgCreateRoom  = "create_room"
	MsgJoinRoom    = "join_room"
	MsgMakeMove    = "make_move"
	MsgPlayAgain   = "play_again"
	MsgSetName     = "set_name"
)

// Message types server → client
const (
	MsgRoomCreated  = "room_created"
	MsgRoomJoined   = "room_joined"
	MsgGameState    = "game_state"
	MsgPlayerLeft   = "player_left"
	MsgError        = "error"
	MsgOpponentName = "opponent_name"
)

type InMessage struct {
	Type        string `json:"type"`
	RoomCode    string `json:"roomCode,omitempty"`
	PlayerName  string `json:"playerName,omitempty"`
	Move        *Move  `json:"move,omitempty"`
	FirstPlayer int    `json:"firstPlayer,omitempty"` // -1 = random
}

type OutMessage struct {
	Type      string     `json:"type"`
	RoomCode  string     `json:"roomCode,omitempty"`
	State     *GameState `json:"state,omitempty"`
	Message   string     `json:"message,omitempty"`
	PlayerIdx int        `json:"playerIdx"` // which seat this client is (0 or 1)
	Name      string     `json:"name,omitempty"`
}

type Room struct {
	mu      sync.Mutex
	code    string
	players [2]*Client // nil if seat empty
	game    GameState
}

func newRoom(code string, p1 *Client, name string) *Room {
	r := &Room{
		code: code,
		game: newGame(code, name),
	}
	r.players[0] = p1
	return r
}

func (r *Room) broadcast(msg OutMessage) {
	data, _ := json.Marshal(msg)
	for idx, p := range r.players {
		if p == nil {
			continue
		}
		m := msg
		m.PlayerIdx = idx
		d, _ := json.Marshal(m)
		p.send <- d
		_ = data
	}
}

func (r *Room) sendTo(idx int, msg OutMessage) {
	if r.players[idx] == nil {
		return
	}
	msg.PlayerIdx = idx
	data, _ := json.Marshal(msg)
	r.players[idx].send <- data
}

func (r *Room) sendState() {
	for idx, p := range r.players {
		if p == nil {
			continue
		}
		msg := OutMessage{
			Type:      MsgGameState,
			State:     &r.game,
			PlayerIdx: idx,
		}
		data, _ := json.Marshal(msg)
		p.send <- data
	}
}

func (r *Room) isEmpty() bool {
	return r.players[0] == nil && r.players[1] == nil
}

// Hub manages all rooms
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]*Room

	register   chan *Client
	unregister chan *Client
	incoming   chan clientMsg
}

type clientMsg struct {
	client *Client
	data   []byte
}

func newHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client, 32),
		unregister: make(chan *Client, 32),
		incoming:   make(chan clientMsg, 256),
	}
}

func (h *Hub) run() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		select {
		case c := <-h.register:
			_ = c // just track via rooms
		case c := <-h.unregister:
			h.handleDisconnect(c)
		case msg := <-h.incoming:
			h.handleMessage(msg.client, msg.data)
		}
	}
}

func (h *Hub) generateCode() string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	h.mu.RLock()
	defer h.mu.RUnlock()
	for {
		b := make([]byte, 5)
		for i := range b {
			b[i] = letters[rand.Intn(len(letters))]
		}
		code := string(b)
		if _, exists := h.rooms[code]; !exists {
			return code
		}
	}
}

func (h *Hub) handleMessage(c *Client, data []byte) {
	var msg InMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("invalid message")
		return
	}

	switch msg.Type {
	case MsgCreateRoom:
		h.createRoom(c, msg.PlayerName)
	case MsgJoinRoom:
		h.joinRoom(c, msg.RoomCode, msg.PlayerName)
	case MsgMakeMove:
		h.makeMove(c, msg.Move)
	case MsgPlayAgain:
		h.playAgain(c, msg.FirstPlayer)
	}
}

func (h *Hub) createRoom(c *Client, name string) {
	if name == "" {
		name = "Player 1"
	}
	code := h.generateCode()
	room := newRoom(code, c, name)
	c.roomCode = code
	c.playerIdx = 0
	c.name = name

	h.mu.Lock()
	h.rooms[code] = room
	h.mu.Unlock()

	data, _ := json.Marshal(OutMessage{
		Type:      MsgRoomCreated,
		RoomCode:  code,
		State:     &room.game,
		PlayerIdx: 0,
	})
	c.send <- data
}

func (h *Hub) joinRoom(c *Client, code string, name string) {
	if name == "" {
		name = "Player 2"
	}
	h.mu.Lock()
	room, ok := h.rooms[code]
	h.mu.Unlock()

	if !ok {
		c.sendError(fmt.Sprintf("Room %s not found", code))
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.players[1] != nil {
		c.sendError("Room is full")
		return
	}

	room.players[1] = c
	room.game.Players[1] = name
	c.roomCode = code
	c.playerIdx = 1
	c.name = name

	room.game.startPlaying(-1) // random first

	// Tell player 2 they joined
	data, _ := json.Marshal(OutMessage{
		Type:      MsgRoomJoined,
		RoomCode:  code,
		State:     &room.game,
		PlayerIdx: 1,
	})
	c.send <- data

	// Tell player 1 opponent joined + new state
	if room.players[0] != nil {
		d, _ := json.Marshal(OutMessage{
			Type:      MsgGameState,
			State:     &room.game,
			PlayerIdx: 0,
		})
		room.players[0].send <- d
	}
}

func (h *Hub) makeMove(c *Client, m *Move) {
	if m == nil {
		c.sendError("missing move")
		return
	}
	h.mu.RLock()
	room, ok := h.rooms[c.roomCode]
	h.mu.RUnlock()
	if !ok {
		c.sendError("not in a room")
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.game.CurrentPlayer != c.playerIdx {
		c.sendError("not your turn")
		return
	}
	if !room.game.applyMove(*m) {
		c.sendError("invalid move")
		return
	}
	room.sendState()
}

func (h *Hub) playAgain(c *Client, firstPlayer int) {
	h.mu.RLock()
	room, ok := h.rooms[c.roomCode]
	h.mu.RUnlock()
	if !ok {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.game.Status != StatusFinished {
		return
	}
	room.game.reset(firstPlayer)
	room.sendState()
}

func (h *Hub) handleDisconnect(c *Client) {
	if c.roomCode == "" {
		return
	}
	h.mu.Lock()
	room, ok := h.rooms[c.roomCode]
	if ok {
		room.mu.Lock()
		room.players[c.playerIdx] = nil
		other := 1 - c.playerIdx
		otherClient := room.players[other]
		room.mu.Unlock()

		if otherClient != nil {
			d, _ := json.Marshal(OutMessage{
				Type:      MsgPlayerLeft,
				Message:   c.name + " left the game",
				PlayerIdx: other,
			})
			otherClient.send <- d
		}

		room.mu.Lock()
		empty := room.isEmpty()
		room.mu.Unlock()
		if empty {
			delete(h.rooms, c.roomCode)
		}
	}
	h.mu.Unlock()
}
