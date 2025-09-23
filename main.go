package main

import (
	"dunExpo/dungeon"
	"dunExpo/game"
	"fmt"
)

func main() {
	fmt.Println("Generating dungeon...")

	newDungeon, floorTiles := dungeon.GenerateDungeon(dungeon.MapWidth, dungeon.MapHeight)

	monsters := game.SpawnMonsters(floorTiles)


	dungeon.PrintDungeon(newDungeon, monsters)

	fmt.Println("Dungeon generation complete.")
}