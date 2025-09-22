package main

import (
	"fmt"
	"math/rand" 
	"time"      
)
const (
	MapWidth  = 80
	MapHeight = 45
)
const (
	TileWall  = 0
	TileFloor = 1
)

func generateDungeon(width, height int) [][]int {
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)

	dungeon := make([][]int, height)
	for i := 0; i < height; i++ {
		dungeon[i] = make([]int, width)
		for j := 0; j < width; j++ {
			dungeon[i][j] = random.Intn(2)
		}
	}
	return dungeon
}

func printDungeon(dungeon [][]int) {
	for y := 0; y < len(dungeon); y++ {
		for x := 0; x < len(dungeon[y]); x++ {
			switch dungeon[y][x] {
			case TileWall:
				fmt.Print("#")
			case TileFloor:
				fmt.Print(".")
			}
		}
		fmt.Println()
	}
}

func main() {
	fmt.Println("Generating a random dungeon map...")
	dungeonMap := generateDungeon(MapWidth, MapHeight)
	printDungeon(dungeonMap)
	fmt.Println("Random map generation complete.")
}