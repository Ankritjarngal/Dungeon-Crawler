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

const logSize = 5

func addMessage(state *game.GameState, message string) {
	state.Log = append([]string{message}, state.Log...)
	if len(state.Log) > logSize {
		state.Log = state.Log[:logSize]
	}
}

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
			case dungeon.TileHealth:
				fmt.Print(dungeon.ColorMagenta + "+" + dungeon.ColorReset)
			}
		}
		fmt.Println()
	}
	fmt.Printf("\nHP: %d/%d | Monsters: %d | Use WASD/Arrows to move, Q/Esc to quit.\n", state.Player.HP, state.Player.MaxHP, len(state.Monsters))
	for _, msg := range state.Log {
		fmt.Println(msg)
	}
}

func main() {
	dungeonMap, floorTiles, startPos, endPos := dungeon.GenerateDungeon(dungeon.MapWidth, dungeon.MapHeight)
	monsters := game.SpawnMonsters(floorTiles)
	player := game.NewPlayer(startPos)

	gameState := game.GameState{
		Dungeon:  dungeonMap,
		Monsters: monsters,
		Player:   player,
		ExitPos:  endPos,
	}

	if err := keyboard.Open(); err != nil {
		log.Fatal(err)
	}
	defer keyboard.Close()

	rand.Seed(time.Now().UnixNano())

	addMessage(&gameState, "Welcome to the dungeon! Find the > to escape.")

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
			damage := player.Attack
			attackedMonster.CurrentHP -= damage
			addMessage(&gameState, fmt.Sprintf("Player attacks the %s for %d damage!", attackedMonster.Template.Name, damage))

			if attackedMonster.CurrentHP > 0 {
				damage = attackedMonster.Template.Attack
				player.HP -= damage
				addMessage(&gameState, fmt.Sprintf("%s attacks Player for %d damage!", attackedMonster.Template.Name, damage))
			} else {
				addMessage(&gameState, fmt.Sprintf("%s is defeated!", attackedMonster.Template.Name))
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

		if gameState.Dungeon[player.Position.Y][player.Position.X] == dungeon.TileHealth {
			healAmount := 10
			player.HP += healAmount
			if player.HP > player.MaxHP {
				player.HP = player.MaxHP
			}
			gameState.Dungeon[player.Position.Y][player.Position.X] = dungeon.TileFloor
			addMessage(&gameState, fmt.Sprintf("You drink from the fountain and recover %d HP.", healAmount))
		}

		if player.Position == gameState.ExitPos {
			render(gameState)
			fmt.Println("\n\nYou have escaped the dungeon! VICTORY!")
			keyboard.GetKey()
			os.Exit(0)
		}

		for _, monster := range gameState.Monsters {
			visionRadius := monster.Template.VisionRadius
			leashRadius := monster.Template.LeashRadius
			distToPlayer := game.Distance(monster.Position, player.Position)
			distToSpawn := game.Distance(monster.Position, monster.SpawnPoint)

			if distToPlayer == 1 {
				damage := monster.Template.Attack
				player.HP -= damage
				addMessage(&gameState, fmt.Sprintf("%s attacks Player for %d damage!", monster.Template.Name, damage))
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