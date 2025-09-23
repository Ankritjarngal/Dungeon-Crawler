package main

import (
	"dunExpo/dungeon"
	"dunExpo/game"
	"fmt"
)

type GameState struct {
	Dungeon  [][]int
	Monsters []*game.Monster
	Player   *game.Player
}

func render(state GameState) {
	monsterMap := make(map[dungeon.Point]*game.Monster)
	for _, m := range state.Monsters {
		monsterMap[m.Position] = m
	}

	for y := 0; y < len(state.Dungeon); y++ {
		for x := 0; x < len(state.Dungeon[y]); x++ {
			currentPoint := dungeon.Point{X: x, Y: y}

			if state.Player.Position == currentPoint {
				fmt.Print(dungeon.ColorCyan + "@" + dungeon.ColorReset)
				continue
			}

			if monster, ok := monsterMap[currentPoint]; ok {
				fmt.Print(monster.Template.Color + string(monster.Template.Rune) + dungeon.ColorReset)
				continue
			}

			switch state.Dungeon[y][x] {
			case dungeon.TileWall:
				fmt.Print(dungeon.ColorGrey + "█" + dungeon.ColorReset)
			case dungeon.TileFloor:
				fmt.Print(dungeon.ColorWhite + "░" + dungeon.ColorReset)
			case dungeon.TileExit:
				fmt.Print(dungeon.ColorYellow + ">" + dungeon.ColorReset)
			}
		}
		fmt.Println()
	}
}
func main() {
	dungeonMap, floorTiles, startPos := dungeon.GenerateDungeon(dungeon.MapWidth, dungeon.MapHeight)
	monsters := game.SpawnMonsters(floorTiles)
	player := game.NewPlayer(startPos)

	gameState := GameState{
		Dungeon:  dungeonMap,
		Monsters: monsters,
		Player:   player,
	}

	
	fmt.Printf("HP: %d/%d | Monsters: %d\n\n", gameState.Player.HP, gameState.Player.MaxHP, len(gameState.Monsters))

	render(gameState)

	fmt.Println("\nmeow")
}