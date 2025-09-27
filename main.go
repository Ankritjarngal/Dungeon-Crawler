package main

import (
	"bufio"
	"dunExpo/dungeon"
	"dunExpo/game"
	"encoding/json"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Server struct {
	Sessions map[string]*Session
	mux      sync.Mutex
}

type Client struct {
	Conn       net.Conn
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
	dungeonMap, floorTiles, _, endPos,itemLocations := dungeon.GenerateDungeon(dungeon.MapWidth, dungeon.MapHeight)
	monsters := game.SpawnMonsters(floorTiles)
	items:=make(map[dungeon.Point]*game.Item)
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

func (s *Server) AddClient(conn net.Conn) {
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

func (s *Session) AddClient(conn net.Conn) {
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
	jsonMsg, _ := json.Marshal(welcomeMsg)
	client.Conn.Write(append(jsonMsg, '\n'))

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
		log.Printf("Player %s has been removed from session.", playerID)
	}
}

func (s *Session) BroadcastState() {
	s.mux.Lock()
	defer s.mux.Unlock()

	stateMsg := map[string]interface{}{"type": "state", "data": s.GameState}
	jsonState, err := json.Marshal(stateMsg)
	if err != nil {
		log.Printf("Error marshalling game state: %v", err)
		return
	}

	for _, client := range s.Clients {
		client.Conn.Write(append(jsonState, '\n'))
	}
}

func (c *Client) Listen() {
	reader := bufio.NewReader(c.Conn)
	for {
		command, err := reader.ReadString('\n')
		if err != nil {
			c.CmdChannel <- ClientCommand{PlayerID: c.PlayerID, Command: "quit"}
			return
		}
		c.CmdChannel <- ClientCommand{PlayerID: c.PlayerID, Command: strings.TrimSpace(command)}
	}
}

func (s *Session) RunLoop() {
	for cmd := range s.CommandStream {
		if cmd.Command == "quit" {
			s.RemoveClient(cmd.PlayerID)
		} else {
			playersWhoWon := game.ProcessPlayerCommand(cmd.PlayerID, cmd.Command, &s.GameState)
			game.UpdateMonsters(&s.GameState)
			if len(playersWhoWon) > 0 {
				s.BroadcastState()
				time.Sleep(100 * time.Millisecond)
				for id := range s.Clients {
					s.RemoveClient(id)
				}
				// TODO: Need a way to remove this session from the main server map.
				return
			}
		}

		s.BroadcastState()
	}
}

func main() {
	server := NewServer()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	log.Println("Game server started on port 8080...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go server.AddClient(conn)
	}
}