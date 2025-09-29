package main

import (
	"dunExpo/dungeon"
	"dunExpo/game"

	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from the Vite React dev server for local development.
		// In production, the origin might be different or not present, so we are permissive.
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" {
			return true
		}
		// Add other allowed origins here for production if needed.
		// For now, let's be flexible.
		if origin == "" {
			return true
		}
		// A more secure check might be needed for a real production environment.
		return true
	},
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
	IsOver        bool
	ExploredTiles map[string]map[dungeon.Point]bool
}

func NewSession() *Session {
	dungeonMap, floorTiles, _, endPos, itemLocations := dungeon.GenerateDungeon(dungeon.MapWidth, dungeon.MapHeight)
	monsters := game.SpawnMonsters(floorTiles)
	items := make(map[dungeon.Point]*game.Item)
	for pos, name := range itemLocations {
		itemTemplate := game.ItemTemplates[name]
		newItem := itemTemplate
		items[pos] = &newItem
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
		ExploredTiles: make(map[string]map[dungeon.Point]bool),
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
		if len(session.Clients) < 4 && !session.IsOver {
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
	s.ExploredTiles[playerID] = make(map[dungeon.Point]bool)
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
		delete(s.GameState.Players, playerID)
		delete(s.ExploredTiles, playerID)
		log.Printf("Player %s has been removed from session.", playerID)
	}
}

func (s *Session) BroadcastState() {
	s.mux.Lock()
	defer s.mux.Unlock()

	teamworkRadius := 6
	teamworkBonusPerAlly := 2
	allVisibleTiles := make(map[string]map[dungeon.Point]bool)
	for playerID, player := range s.GameState.Players {
		if player.Status != "playing" {
			continue
		}
		nearbyAllyCount := 0
		for otherPlayerID, otherPlayer := range s.GameState.Players {
			if playerID != otherPlayerID && otherPlayer.Status == "playing" {
				if game.Distance(player.Position, otherPlayer.Position) <= teamworkRadius {
					nearbyAllyCount++
				}
			}
		}
		effectiveVision := player.VisionRadius + (nearbyAllyCount * teamworkBonusPerAlly)
		allVisibleTiles[playerID] = game.CalculateVisibility(player.Position, effectiveVision)
	}

	for playerID := range s.GameState.Players {
		if visible, ok := allVisibleTiles[playerID]; ok {
			for point := range visible {
				s.ExploredTiles[playerID][point] = true
			}
		}
	}

	for _, client := range s.Clients {
		player, ok := s.GameState.Players[client.PlayerID]
		if !ok {
			continue
		}
		visibleTilesMap := allVisibleTiles[client.PlayerID]
		exploredTilesMap := s.ExploredTiles[client.PlayerID]
		visibleForJSON := []dungeon.Point{}
		for p := range visibleTilesMap {
			visibleForJSON = append(visibleForJSON, p)
		}
		exploredForJSON := []dungeon.Point{}
		for p := range exploredTilesMap {
			exploredForJSON = append(exploredForJSON, p)
		}
		itemsForJSON := []game.ItemOnGroundJSON{}
		for pos, item := range s.GameState.ItemsOnGround {
			itemsForJSON = append(itemsForJSON, game.ItemOnGroundJSON{Position: pos, Item: item})
		}
		highlighted := []dungeon.Point{}
		if player.Status == "targeting" && player.Target != nil {
			highlighted = game.GetLineOfSightPath(player.Position, *player.Target)
		}
		stateForJSON := game.GameStateForJSON{
			Dungeon:          s.GameState.Dungeon,
			Monsters:         s.GameState.Monsters,
			Players:          s.GameState.Players,
			ExitPos:          s.GameState.ExitPos,
			Log:              s.GameState.Log,
			ItemsOnGround:    itemsForJSON,
			HighlightedTiles: highlighted,
			VisibleTiles:     visibleForJSON,
		}
		stateMsg := map[string]interface{}{"type": "state", "data": stateForJSON}
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
		} else {
			playersWhoWon, endTurnEarly := game.ProcessPlayerCommand(cmd.PlayerID, cmd.Command, &s.GameState)
			if !endTurnEarly {
				game.UpdateMonsters(&s.GameState)
			}
			if len(playersWhoWon) > 0 {
				s.BroadcastState()
				time.Sleep(100 * time.Millisecond)
				for id := range s.Clients {
					s.RemoveClient(id)
				}
				s.mux.Lock()
				s.IsOver = true
				s.mux.Unlock()
				return
			}
		}
		s.BroadcastState()
	}
}

func main() {
	server := NewServer()
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", server.handleWebSocketConnections)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Game server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}