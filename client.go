package main

import (
	"bufio"
	"dunExpo/dungeon"
	"dunExpo/game"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/eiannone/keyboard"
)

var selfID string

func render(state game.GameStateForJSON) {
	fmt.Print("\033[H\033[2J")

	itemMap := make(map[dungeon.Point]*game.Item)
	for _, itemOnGround := range state.ItemsOnGround {
		itemMap[itemOnGround.Position] = itemOnGround.Item
	}

	monsterMap := make(map[dungeon.Point]*game.Monster)
	for _, m := range state.Monsters {
		monsterMap[m.Position] = m
	}

	playerMap := make(map[dungeon.Point]*game.Player)
	for _, p := range state.Players {
		playerMap[p.Position] = p
	}

	var selfPlayer *game.Player
	if p, ok := state.Players[selfID]; ok {
		selfPlayer = p
	}

	for y := 0; y < len(state.Dungeon); y++ {
		for x := 0; x < len(state.Dungeon[y]); x++ {
			currentPoint := dungeon.Point{X: x, Y: y}

			if player, ok := playerMap[currentPoint]; ok {
				runeToDraw := "@"
				colorToUse := dungeon.ColorCyan
				if player.Status == "defeated" {
					runeToDraw = "%"
					colorToUse = dungeon.ColorGrey
				} else if player.ID != selfID {
					runeToDraw = "P"
				}
				fmt.Print(colorToUse + runeToDraw + dungeon.ColorReset)
				continue
			}

			if item, ok := itemMap[currentPoint]; ok {
				fmt.Print(item.Color + string(item.Rune) + dungeon.ColorReset)
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

	if selfPlayer != nil {
		status := fmt.Sprintf("HP: %d/%d", selfPlayer.HP, selfPlayer.MaxHP)
		if selfPlayer.EquippedWeapon != nil {
			status += fmt.Sprintf(" | Weapon: %s (%d dmg)", selfPlayer.EquippedWeapon.Name, selfPlayer.EquippedWeapon.Damage)
		}
		fmt.Printf("\n%s | Monsters: %d | Players: %d\n", status, len(state.Monsters), len(state.Players))
	} else {
		fmt.Println("\nConnecting...")
	}

	for i := len(state.Log) - 1; i >= 0; i-- {
		fmt.Println(state.Log[i])
	}
}

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	welcomeMsgBytes, err := reader.ReadBytes('\n')
	if err != nil {
		log.Fatalf("Did not receive welcome message: %v", err)
	}

	var welcomeMsg map[string]string
	json.Unmarshal(welcomeMsgBytes, &welcomeMsg)
	if welcomeMsg["type"] == "welcome" {
		selfID = welcomeMsg["id"]
	} else {
		log.Fatalf("Expected welcome message, got: %s", string(welcomeMsgBytes))
	}

	log.Printf("Connected to server. You are Player %s.", selfID)

	go func() {
		for {
			jsonMsg, err := reader.ReadBytes('\n')
			if err != nil {
				log.Println("Server connection lost.")
				os.Exit(0)
			}
			var msg map[string]json.RawMessage
			if err := json.Unmarshal(jsonMsg, &msg); err != nil {
				continue
			}
			if t, ok := msg["type"]; ok && string(t) == "\"state\"" {
				var gameState game.GameStateForJSON
				if err := json.Unmarshal(msg["data"], &gameState); err != nil {
					log.Printf("Error unmarshalling game state: %v", err)
					continue
				}
				render(gameState)
			}
		}
	}()

	if err := keyboard.Open(); err != nil {
		log.Fatal(err)
	}
	defer keyboard.Close()

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			log.Fatal(err)
		}
		command := ""
		switch {
		case char == 'w' || key == keyboard.KeyArrowUp:
			command = "w"
		case char == 'a' || key == keyboard.KeyArrowLeft:
			command = "a"
		case char == 's' || key == keyboard.KeyArrowDown:
			command = "s"
		case char == 'd' || key == keyboard.KeyArrowRight:
			command = "d"
		case char == 'g':
			command = "g"
		case char == 'q' || key == keyboard.KeyEsc:
			os.Exit(0)
		}
		if command != "" {
			fmt.Fprintf(conn, "%s\n", command)
		}
	}
}