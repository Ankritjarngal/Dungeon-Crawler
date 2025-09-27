package main

import (
	"dunExpo/dungeon"
	"dunExpo/game"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Server struct {
	Sessions map[string]*Session
	mux      sync.Mutex
}

type Client struct {
	Conn       *websocket.Conn
	PlayerID   string
	CmdChannel chan<- ClientCommand
}

type ClientCommand struct {
	PlayerID string
	Command  string
}

type Session struct {
	GameState     game.GameState
	Clients       map[string]*Client
	mux           sync.Mutex
	CommandStream chan ClientCommand
}

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
	}
}

func NewServer() *Server {
	return &Server{
		Sessions: make(map[string]*Session),
	}
}

func (s *Server) handleWebSocketConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	s.AddClient(ws)
}

func (s *Server) AddClient(conn *websocket.Conn) {
	s.mux.Lock()
	defer s.mux.Unlock()
	for _, session := range s.Sessions {
		if len(session.Clients) < 5 {
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

func (s *Session) RemoveClient(playerID string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if client, ok := s.Clients[playerID]; ok {
		client.Conn.Close()
		delete(s.Clients, playerID)
		log.Printf("Player %s connection closed.", playerID)
	}
}




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

	// THE FIX: We create our final message...
	stateMsg := map[string]interface{}{"type": "state", "data": stateForJSON}

	// ...and then we loop and send it using the correct helper function.
	for _, client := range s.Clients {
		if err := client.Conn.WriteJSON(stateMsg); err != nil {
			log.Printf("broadcast error to %s: %v", client.PlayerID, err)
		}
	}
}

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


func (s *Session) RunLoop() {
	for cmd := range s.CommandStream {
		if cmd.Command == "quit" {
			s.RemoveClient(cmd.PlayerID)
			delete(s.GameState.Players, cmd.PlayerID)
			s.BroadcastState()
			continue
		}

		playersWhoWon, endTurnEarly := game.ProcessPlayerCommand(cmd.PlayerID, cmd.Command, &s.GameState)

		if !endTurnEarly {
			game.UpdateMonsters(&s.GameState)
		}

		s.BroadcastState()

		if len(playersWhoWon) > 0 {
			time.Sleep(100 * time.Millisecond)
			for id := range s.Clients {
				s.RemoveClient(id)
			}
			return
		}
	}
}
func main() {
	server := NewServer()
	http.HandleFunc("/ws", server.handleWebSocketConnections)
	log.Println("Game server starting on ws://localhost:8080/ws")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}