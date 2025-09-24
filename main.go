package main

import (
	"dunExpo/dungeon"
	"dunExpo/game"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/eiannone/keyboard"
)

func render(state game.GameState) {
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

	gameState := game.GameState{
		Dungeon:  dungeonMap,
		Monsters: monsters,
		Player:   player,
	}

	if err := keyboard.Open(); err != nil {
		log.Fatal(err)
	}
	defer keyboard.Close()

	rand.Seed(time.Now().UnixNano())

	for {
		render(gameState)

		char, key, err := keyboard.GetKey()
		if err != nil {
			log.Fatal(err)
		}

		var attackedMonster *game.Monster
		switch {
		case char == 'w' || key == keyboard.KeyArrowUp:
			attackedMonster = player.Move(0, -1, &gameState)
		case char == 'a' || key == keyboard.KeyArrowLeft:
			attackedMonster = player.Move(-1, 0, &gameState)
		case char == 's' || key == keyboard.KeyArrowDown:
			attackedMonster = player.Move(0, 1, &gameState)
		case char == 'd' || key == keyboard.KeyArrowRight:
			attackedMonster = player.Move(1, 0, &gameState)
		case char == 'q' || key == keyboard.KeyEsc:
			os.Exit(0)
		}

		if attackedMonster != nil {
			attackedMonster.CurrentHP -= player.Attack
			if attackedMonster.CurrentHP > 0 {
				player.HP -= attackedMonster.Template.Attack
			}
		}

		var survivingMonsters []*game.Monster
		for _, m := range gameState.Monsters {
			if m.CurrentHP > 0 {
				survivingMonsters = append(survivingMonsters, m)
			}
		}
		gameState.Monsters = survivingMonsters

		if player.HP <= 0 {
			render(gameState)
			fmt.Println("\n\nYou have been defeated. GAME OVER.")
			keyboard.GetKey()
			os.Exit(0)
		}

		for _, monster := range gameState.Monsters {
			visionRadius := monster.Template.VisionRadius
			leashRadius := monster.Template.LeashRadius
			distToPlayer := game.Distance(monster.Position, player.Position)
			distToSpawn := game.Distance(monster.Position, monster.SpawnPoint)

			if distToPlayer == 1 {
				player.HP -= monster.Template.Attack
				continue
			}

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
					monster.Move(dx, 0, &gameState)
				} else {
					monster.Move(0, dy, &gameState)
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
					monster.Move(dx, 0, &gameState)
				} else {
					monster.Move(0, dy, &gameState)
				}
			} else {
				direction := rand.Intn(4)
				switch direction {
				case 0:
					monster.Move(0, -1, &gameState)
				case 1:
					monster.Move(0, 1, &gameState)
				case 2:
					monster.Move(-1, 0, &gameState)
				case 3:
					monster.Move(1, 0, &gameState)
				}
			}
		}
	}
}