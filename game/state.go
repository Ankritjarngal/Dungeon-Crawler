package game

import (
	"dunExpo/dungeon"
	"math/rand"
)

// GameState is the internal, high-performance representation of the game world.
// It uses a map for ItemsOnGround for fast lookups on the server.
type GameState struct {
	Dungeon       [][]int
	Monsters      []*Monster
	Players       map[string]*Player
	ExitPos       dungeon.Point
	Log           []string
	ItemsOnGround map[dungeon.Point]*Item
}

// GameStateForJSON is a "shipping manifest" used only for sending data to the client.
// It uses a slice for items because JSON keys must be strings.
type GameStateForJSON struct {
	Dungeon       [][]int
	Monsters      []*Monster
	Players       map[string]*Player
	ExitPos       dungeon.Point
	Log           []string
	ItemsOnGround []ItemOnGroundJSON
	HighlightedTiles []dungeon.Point 
	VisibleTiles []dungeon.Point
	PlayerTrails map[string][]dungeon.Point
}

// ItemOnGroundJSON represents a single item on the ground for sending.
type ItemOnGroundJSON struct {
	Position dungeon.Point
	Item     *Item
}

func (gs *GameState) GetRandomSpawnPoint() dungeon.Point {
	var floorTiles []dungeon.Point
	for y, row := range gs.Dungeon {
		for x, tile := range row {
			if tile == dungeon.TileFloor {
				isOccupied := false
				for _, p := range gs.Players {
					if p.Position.X == x && p.Position.Y == y {
						isOccupied = true
						break
					}
				}
				if !isOccupied {
					floorTiles = append(floorTiles, dungeon.Point{X: x, Y: y})
				}
			}
		}
	}
	if len(floorTiles) > 0 {
		return floorTiles[rand.Intn(len(floorTiles))]
	}
	return dungeon.Point{}
}

const logSize = 5

func (gs *GameState) AddMessage(message string) {
	gs.Log = append([]string{message}, gs.Log...)
	if len(gs.Log) > logSize {
		gs.Log = gs.Log[:logSize]
	}
}