package main

import (
	"dunExpo/dungeon"
	"dunExpo/game"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/eiannone/keyboard"
)

type GameState struct {
	Dungeon  [][]int
	Monsters []*game.Monster
	Player   *game.Player
}

func render(state GameState) {
	fmt.Print("\033[H\033[2J")

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
	fmt.Printf("\nHP: %d/%d | Monsters: %d | Use WASD/Arrows to move, Q/Esc to quit.\n", state.Player.HP, state.Player.MaxHP, len(state.Monsters))
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

	if err := keyboard.Open(); err != nil {
		log.Fatal(err)
	}
	defer keyboard.Close()

	for {
		render(gameState)

		char, key, err := keyboard.GetKey()
		if err != nil {
			log.Fatal(err)
		}

		switch {
		case char == 'w' || key == keyboard.KeyArrowUp:
			player.Move(0, -1, gameState.Dungeon)
		case char == 'a' || key == keyboard.KeyArrowLeft:
			player.Move(-1, 0, gameState.Dungeon)
		case char == 's' || key == keyboard.KeyArrowDown:
			player.Move(0, 1, gameState.Dungeon)
		case char == 'd' || key == keyboard.KeyArrowRight:
			player.Move(1, 0, gameState.Dungeon)
		case char == 'q' || key == keyboard.KeyEsc:
			os.Exit(0)
		}

		for _, monster := range gameState.Monsters {
			visionRadius := monster.Template.VisionRadius
			leashRadius := monster.Template.LeashRadius

			distToPlayer := game.Distance(monster.Position, player.Position)
			distToSpawn := game.Distance(monster.Position, monster.SpawnPoint)

			if distToPlayer <= visionRadius && distToSpawn < leashRadius {
				dx, dy := 0, 0
				if player.Position.X > monster.Position.X {
					dx = 1
				} else if player.Position.X < monster.Position.X {
					dx = -1
				}
				if player.Position.Y > monster.Position.Y {
					dy = 1
				} else if player.Position.Y < monster.Position.Y {
					dy = -1
				}
				if rand.Intn(2) == 0 {
					monster.Move(dx, 0, gameState.Dungeon)
				} else {
					monster.Move(0, dy, gameState.Dungeon)
				}
			} else if distToSpawn > 0 {
				dx, dy := 0, 0
				if monster.SpawnPoint.X > monster.Position.X {
					dx = 1
				} else if monster.SpawnPoint.X < monster.Position.X {
					dx = -1
				}
				if monster.SpawnPoint.Y > monster.Position.Y {
					dy = 1
				} else if monster.SpawnPoint.Y < monster.Position.Y {
					dy = -1
				}
				if rand.Intn(2) == 0 {
					monster.Move(dx, 0, gameState.Dungeon)
				} else {
					monster.Move(0, dy, gameState.Dungeon)
				}
			} else {
				direction := rand.Intn(4)
				switch direction {
				case 0:
					monster.Move(0, -1, gameState.Dungeon)
				case 1:
					monster.Move(0, 1, gameState.Dungeon)
				case 2:
					monster.Move(-1, 0, gameState.Dungeon)
				case 3:
					monster.Move(1, 0, gameState.Dungeon)
				}
			}
		}
	}
}