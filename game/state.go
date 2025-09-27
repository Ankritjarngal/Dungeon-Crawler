package game

import (
	"dunExpo/dungeon"
	"math/rand"
)

type GameState struct {
	Dungeon  [][]int
	Monsters []*Monster
	Players  map[string]*Player
	ExitPos  dungeon.Point
	Log      []string
	ItemsOnGround map[dungeon.Point]*Item
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