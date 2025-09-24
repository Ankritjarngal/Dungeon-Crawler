package main

import (
	"bufio"
	"dunExpo/dungeon"
	"dunExpo/game"
	"encoding/json"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	Conn       net.Conn
	PlayerID   string
	CmdChannel chan<- ClientCommand
}

type ClientCommand struct {
	PlayerID string
	Command  string
}

type Game struct {
	GameState     game.GameState
	Clients       map[string]*Client
	mux           sync.Mutex
	CommandStream chan ClientCommand
}

func NewGame() *Game {
	dungeonMap, floorTiles, _, endPos := dungeon.GenerateDungeon(dungeon.MapWidth, dungeon.MapHeight)
	monsters := game.SpawnMonsters(floorTiles)

	gs := game.GameState{
		Dungeon:  dungeonMap,
		Monsters: monsters,
		Players:  make(map[string]*game.Player),
		ExitPos:  endPos,
	}

	return &Game{
		GameState:     gs,
		Clients:       make(map[string]*Client),
		CommandStream: make(chan ClientCommand),
	}
}

func (g *Game) AddClient(conn net.Conn) {
	g.mux.Lock()
	defer g.mux.Unlock()

	playerID := uuid.New().String()
	startPos := g.GameState.GetRandomSpawnPoint()
	newPlayer := game.NewPlayer(playerID, startPos)
	g.GameState.Players[playerID] = newPlayer

	client := &Client{
		Conn:       conn,
		PlayerID:   playerID,
		CmdChannel: g.CommandStream,
	}
	g.Clients[playerID] = client

	welcomeMsg := map[string]string{"type": "welcome", "id": playerID}
	jsonMsg, _ := json.Marshal(welcomeMsg)
	client.Conn.Write(append(jsonMsg, '\n'))

	log.Printf("Player %s (%s) has joined.", playerID, conn.RemoteAddr())
	go client.Listen()
}

func (g *Game) RemoveClient(playerID string) {
	g.mux.Lock()
	defer g.mux.Unlock()

	if client, ok := g.Clients[playerID]; ok {
		client.Conn.Close()
		delete(g.Clients, playerID)
		delete(g.GameState.Players, playerID)
		log.Printf("Player %s has been removed from the game.", playerID)
	}
}

func (g *Game) BroadcastState() {
	g.mux.Lock()
	defer g.mux.Unlock()

	stateMsg := map[string]interface{}{"type": "state", "data": g.GameState}
	jsonState, err := json.Marshal(stateMsg)
	if err != nil {
		log.Printf("Error marshalling game state: %v", err)
		return
	}

	for _, client := range g.Clients {
		client.Conn.Write(append(jsonState, '\n'))
	}
}

func (c *Client) Listen() {
	reader := bufio.NewReader(c.Conn)
	for {
		command, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Printf("Player %s connection closed.", c.PlayerID)
			}
			c.CmdChannel <- ClientCommand{PlayerID: c.PlayerID, Command: "quit"}
			return
		}
		c.CmdChannel <- ClientCommand{PlayerID: c.PlayerID, Command: strings.TrimSpace(command)}
	}
}

func (g *Game) RunLoop() {
	for cmd := range g.CommandStream {
		if cmd.Command == "quit" {
			g.RemoveClient(cmd.PlayerID)
		} else {
			playersToRemoveFromPlayerTurn := game.ProcessPlayerCommand(cmd.PlayerID, cmd.Command, &g.GameState)
			playersToRemoveFromMonsterTurn := game.UpdateMonsters(&g.GameState)

			for id := range playersToRemoveFromPlayerTurn {
				g.RemoveClient(id)
			}
			for id := range playersToRemoveFromMonsterTurn {
				g.RemoveClient(id)
			}
		}

		g.BroadcastState()
	}
}

func main() {
	game := NewGame()
	go game.RunLoop()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	log.Println("Game server started on port 8080...")
	log.Printf("It is currently %s in Srinagar.", time.Now().Format("3:04 PM"))


	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go game.AddClient(conn)
	}
}