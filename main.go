package main

import (
	"dunExpo/dungeon"
	"dunExpo/game"
	"log"
	"net/http"
	"sync"
	"time"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from the Vite React dev server.
		// In a real production app, you would make this stricter.
		return r.Header.Get("Origin") == "http://localhost:5173"
	},
}

// Server is the top-level object that manages all game sessions.
type Server struct {
	Sessions map[string]*Session
	mux      sync.Mutex
}

// Client represents a single connected player.
type Client struct {
	Conn       *websocket.Conn
	PlayerID   string
	CmdChannel chan<- ClientCommand
}

// ClientCommand is a message from a client to be processed by a session's game loop.
type ClientCommand struct {
	PlayerID string
	Command  string
}

// Session manages a single, independent game world with up to 5 players.
type Session struct {
	GameState     game.GameState
	Clients       map[string]*Client
	mux           sync.Mutex
	CommandStream chan ClientCommand
	IsOver        bool
}

// NewSession creates a new, fully generated game world.
func NewSession() *Session {
	dungeonMap, floorTiles, _, endPos, itemLocations := dungeon.GenerateDungeon(dungeon.MapWidth, dungeon.MapHeight)
	monsters := game.SpawnMonsters(floorTiles)
	items := make(map[dungeon.Point]*game.Item)
	for pos, name := range itemLocations {
		itemTemplate := game.ItemTemplates[name]
		items[pos] = &itemTemplate
	}
	gs := game.GameState{
		Dungeon:       dungeonMap,
		Monsters:      monsters,
		Players:       make(map[string]*game.Player),
		ExitPos:       endPos,
		ItemsOnGround: items,
	}
	return &Session{
		GameState:     gs,
		Clients:       make(map[string]*Client),
		CommandStream: make(chan ClientCommand),
		IsOver:        false,
	}
}

// NewServer creates the main server instance.
func NewServer() *Server {
	return &Server{
		Sessions: make(map[string]*Session),
	}
}

// handleWebSocketConnections is the entry point for all new connections.
func (s *Server) handleWebSocketConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	s.AddClient(ws)
}

// AddClient is the server's "bouncer" logic. It finds a session or creates a new one.
func (s *Server) AddClient(conn *websocket.Conn) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, session := range s.Sessions {
		if len(session.Clients) < 5 && !session.IsOver {
			log.Printf("Player %s is joining an existing session.", conn.RemoteAddr())
			session.AddClient(conn)
			session.BroadcastState()
			return
		}
	}

	log.Printf("Creating a new session for player %s.", conn.RemoteAddr())
	sessionID := uuid.New().String()[0:8]
	newSession := NewSession()
	s.Sessions[sessionID] = newSession

	go newSession.RunLoop()
	newSession.AddClient(conn)
	newSession.BroadcastState()
}

// AddClient adds a player to a specific session.
func (s *Session) AddClient(conn *websocket.Conn) {
	s.mux.Lock()
	defer s.mux.Unlock()

	playerID := uuid.New().String()
	startPos := s.GameState.GetRandomSpawnPoint()
	newPlayer := game.NewPlayer(playerID, startPos)
	s.GameState.Players[playerID] = newPlayer

	client := &Client{
		Conn:       conn,
		PlayerID:   playerID,
		CmdChannel: s.CommandStream,
	}
	s.Clients[playerID] = client

	welcomeMsg := map[string]string{"type": "welcome", "id": playerID}
	conn.WriteJSON(welcomeMsg)

	log.Printf("Player %s (%s) has joined session.", playerID, conn.RemoteAddr())
	go client.Listen()
}

// RemoveClient closes a player's connection and removes them from a session.
func (s *Session) RemoveClient(playerID string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if client, ok := s.Clients[playerID]; ok {
		client.Conn.Close()
		delete(s.Clients, playerID)
		log.Printf("Player %s connection closed.", playerID)
	}
}

// BroadcastState sends the current game state to all players in a session.
func (s *Session) BroadcastState() {
	s.mux.Lock()
	defer s.mux.Unlock()

	itemsForJSON := []game.ItemOnGroundJSON{}
	for pos, item := range s.GameState.ItemsOnGround {
		itemsForJSON = append(itemsForJSON, game.ItemOnGroundJSON{Position: pos, Item: item})
	}

	highlighted := []dungeon.Point{}
	for _, p := range s.GameState.Players {
		if p.Status == "targeting" && p.Target != nil {
			highlighted = game.GetLineOfSightPath(p.Position, *p.Target)
			break
		}
	}

	stateForJSON := game.GameStateForJSON{
		Dungeon:          s.GameState.Dungeon,
		Monsters:         s.GameState.Monsters,
		Players:          s.GameState.Players,
		ExitPos:          s.GameState.ExitPos,
		Log:              s.GameState.Log,
		ItemsOnGround:    itemsForJSON,
		HighlightedTiles: highlighted,
	}

	stateMsg := map[string]interface{}{"type": "state", "data": stateForJSON}
	for _, client := range s.Clients {
		if err := client.Conn.WriteJSON(stateMsg); err != nil {
			log.Printf("broadcast error to %s: %v", client.PlayerID, err)
		}
	}
}

// Listen reads commands from a client's connection and sends them to the game loop.
func (c *Client) Listen() {
	defer func() {
		c.CmdChannel <- ClientCommand{PlayerID: c.PlayerID, Command: "quit"}
	}()
	for {
		_, p, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("read error for %s: %v", c.PlayerID, err)
			break
		}
		c.CmdChannel <- ClientCommand{PlayerID: c.PlayerID, Command: string(p)}
	}
}

// RunLoop is the turn-based game loop for a single session.
func (s *Session) RunLoop() {
	for cmd := range s.CommandStream {
		if cmd.Command == "quit" {
			s.RemoveClient(cmd.PlayerID)
			delete(s.GameState.Players, cmd.PlayerID)
		} else {
			playersWhoWon, endTurnEarly := game.ProcessPlayerCommand(cmd.PlayerID, cmd.Command, &s.GameState)
			if !endTurnEarly {
				game.UpdateMonsters(&s.GameState)
			}
			if len(playersWhoWon) > 0 {
				s.mux.Lock()
				s.IsOver = true
				s.mux.Unlock()

				s.BroadcastState()
				time.Sleep(100 * time.Millisecond)
				for id := range s.Clients {
					s.RemoveClient(id)
				}
				return
			}
		}
		s.BroadcastState()
	}
}

// main is the entry point for the entire application.
func main() {
	server := NewServer()
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", server.handleWebSocketConnections)

	log.Println("Game server starting on http://localhost:8080")
	// You will need to add "os" to your imports at the top of main.go
port := os.Getenv("PORT")
if port == "" {
    port = "8080" // Default for local running
}

log.Printf("Game server starting on port %s", port)
if err := http.ListenAndServe(":"+port, nil); err != nil {
    log.Fatal("ListenAndServe:", err)
}
}