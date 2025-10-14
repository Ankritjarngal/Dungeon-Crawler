package main

import (
	"dunExpo/dungeon"
	"dunExpo/game"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {

		return true
	},
}

type InitialMessage struct {
	Type string `json:"type"`
	Code string `json:"code,omitempty"`
}

type ServerResponse struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	ID      string `json:"id,omitempty"`
	Code    string `json:"code,omitempty"`
	Result  string `json:"result,omitempty"`
}

type Server struct {
	Sessions map[string]*Session
	mux      sync.Mutex
	cleanup  chan string
}

type Client struct {
	Conn       *websocket.Conn
	PlayerID   string
	CmdChannel chan<- game.ClientCommand
}

type Session struct {
	Code          string
	GameState     game.GameState
	Clients       map[string]*Client
	mux           sync.Mutex
	CommandStream chan game.ClientCommand
	IsOver        bool
	cleanup       chan<- string
}

func (s *Session) BroadcastGameOver(result string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	gameOverMsg := ServerResponse{
		Type:   "gameOver",
		Result: result,
	}
	for _, client := range s.Clients {
		client.Conn.WriteJSON(gameOverMsg)
	}
}
func generateRoomCode() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 4)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

func NewSession(code string, cleanup chan<- string) *Session {
	dungeonMap, floorTiles, _, endPos, itemLocations := dungeon.GenerateDungeon(dungeon.MapWidth, dungeon.MapHeight)
	monsters := game.SpawnMonsters(floorTiles,endPos)
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
		Code:          code,
		GameState:     gs,
		Clients:       make(map[string]*Client),
		CommandStream: make(chan game.ClientCommand, 100),
		IsOver:        false,
		cleanup:       cleanup,
	}
}

func NewServer() *Server {
	return &Server{
		Sessions: make(map[string]*Session),
		cleanup:  make(chan string, 10),
	}
}

func (s *Server) RunCleanupLoop() {
	for code := range s.cleanup {
		s.mux.Lock()
		delete(s.Sessions, code)
		log.Printf("Cleaned up and removed session %s.", code)
		s.mux.Unlock()
	}
}

func (s *Server) handleWebSocketConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	var msg InitialMessage
	if err := ws.ReadJSON(&msg); err != nil {
		log.Printf("error reading initial message: %v", err)
		ws.Close()
		return
	}
	s.mux.Lock()
	var session *Session
	var ok bool
	switch msg.Type {
	case "create":
		code := strings.ToUpper(msg.Code)
		if _, exists := s.Sessions[code]; exists {
			ws.WriteJSON(ServerResponse{Type: "error", Message: "Room code is already taken."})
			s.mux.Unlock()
			ws.Close()
			return
		}
		session = NewSession(code, s.cleanup)
		s.Sessions[code] = session
		go session.RunLoop()
		log.Printf("New session created with code: %s", code)
	case "join":
		code := strings.ToUpper(msg.Code)
		session, ok = s.Sessions[code]
		if !ok || session.IsOver {
			ws.WriteJSON(ServerResponse{Type: "error", Message: "Room not found or has ended."})
			s.mux.Unlock()
			ws.Close()
			return
		}
		if len(session.Clients) >= 5 {
			ws.WriteJSON(ServerResponse{Type: "error", Message: "Room is full."})
			s.mux.Unlock()
			ws.Close()
			return
		}
	default:
		ws.WriteJSON(ServerResponse{Type: "error", Message: "Invalid request."})
		s.mux.Unlock()
		ws.Close()
		return
	}
	s.mux.Unlock()
	session.AddClient(ws)
}

func (s *Session) AddClient(conn *websocket.Conn) {
	playerID := uuid.New().String()
	s.mux.Lock()
	startPos := s.GameState.GetRandomSpawnPoint()
	newPlayer := game.NewPlayer(playerID, startPos)
	s.GameState.Players[playerID] = newPlayer
	client := &Client{
		Conn:       conn,
		PlayerID:   playerID,
		CmdChannel: s.CommandStream,
	}
	s.Clients[playerID] = client
	s.mux.Unlock()
	conn.WriteJSON(ServerResponse{Type: "welcome", ID: playerID, Code: s.Code})
	log.Printf("Player %s (%s) has joined session %s.", playerID, conn.RemoteAddr(), s.Code)
	go client.Listen(s)
	s.BroadcastState()
}

func (s *Session) RemoveClient(playerID string, shouldCloseConn bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if client, ok := s.Clients[playerID]; ok {
		if shouldCloseConn {
			client.Conn.Close()
		}
		delete(s.Clients, playerID)
		delete(s.GameState.Players, playerID)
		log.Printf("Player %s removed from session %s.", playerID, s.Code)
	}
}

func (s *Session) BroadcastState() {
	s.mux.Lock()
	defer s.mux.Unlock()
	for _, client := range s.Clients {
		player, ok := s.GameState.Players[client.PlayerID]
		if !ok {
			continue
		}
		teamworkRadius := 8
		teamworkBonusPerAlly := 2
		nearbyAllyCount := 0
		if player.Status == "playing" {
			for otherPlayerID, otherPlayer := range s.GameState.Players {
				if player.ID != otherPlayerID && otherPlayer.Status == "playing" {
					if game.Distance(player.Position, otherPlayer.Position) <= teamworkRadius {
						nearbyAllyCount++
					}
				}
			}
		}
		effectiveVision := player.VisionRadius + (nearbyAllyCount * teamworkBonusPerAlly)
		visibleTilesMap := game.CalculateVisibility(player.Position, effectiveVision)
		visibleForJSON := []dungeon.Point{}
		for p := range visibleTilesMap {
			visibleForJSON = append(visibleForJSON, p)
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
			log.Printf("[ERROR] broadcast error to %s: %v", client.PlayerID[0:4], err)
		}
	}
}

func (c *Client) Listen(s *Session) {
	defer func() {
		log.Printf("[DEBUG] Client %s Listen() ending, sending quit", c.PlayerID[0:4])
		s.CommandStream <- game.ClientCommand{PlayerID: c.PlayerID, Command: "quit"}
	}()
	for {
		_, p, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("[DEBUG] ReadMessage error for %s: %v", c.PlayerID[0:4], err)
			break
		}
		s.CommandStream <- game.ClientCommand{PlayerID: c.PlayerID, Command: string(p)}
	}
}

func (s *Session) RunLoop() {
    defer func() {
        log.Printf("Session %s RunLoop ended.", s.Code)
        s.cleanup <- s.Code
    }()

    for cmd := range s.CommandStream {
        if cmd.Command == "quit" {
            s.RemoveClient(cmd.PlayerID, true)
            if len(s.Clients) == 0 {
                log.Printf("Session %s is empty, closing.", s.Code)
                s.mux.Lock()
                s.IsOver = true
                s.mux.Unlock()
                return
            }
        } else {
            playersWhoWon, endTurnEarly := game.ProcessPlayerCommand(cmd.PlayerID, cmd.Command, &s.GameState)
            if !endTurnEarly {
                game.UpdateMonsters(&s.GameState)
            }
            allPlayersDefeated := len(s.GameState.Players) > 0
            for _, p := range s.GameState.Players {
                if p.Status == "playing" || p.Status == "targeting" {
                    allPlayersDefeated = false
                    break
                }
            }
            if len(playersWhoWon) > 0 || allPlayersDefeated {
                var result string
                if allPlayersDefeated {
                    result = "defeat"
                    s.GameState.AddMessage("All players have been defeated! The dungeon claims its victims.")
                } else {
                    result = "victory"
                }
                s.BroadcastState()
                time.Sleep(100 * time.Millisecond)
                s.BroadcastGameOver(result)
                time.Sleep(100 * time.Millisecond)
                s.mux.Lock()
                s.IsOver = true
                s.mux.Unlock()
                log.Printf("Session %s has ended. The loop will now terminate, leaving connections open.", s.Code)
                return
            }
        }
        s.BroadcastState()
    }
}

func main() {
	server := NewServer()
	go server.RunCleanupLoop()
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

