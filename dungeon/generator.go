package dungeon

import (
	"math/rand"
	"time"
)

const (
	MapWidth   = 80
	MapHeight  = 45
	TileWall   = 0
	TileFloor  = 1
	TileExit   = 2
	TileHealth = 3
)

const (
	ColorGrey    = "\x1b[90m"
	ColorWhite   = "\x1b[37m"
	ColorYellow  = "\x1b[93m"
	ColorGreen   = "\x1b[92m"
	ColorRed     = "\x1b[91m"
	ColorCyan    = "\x1b[36m"
	ColorMagenta = "\x1b[95m"
	ColorReset   = "\x1b[0m"
)

type Point struct{ X, Y int }

func GenerateDungeon(width, height int) ([][]int, []Point, Point, Point, map[Point]string) {
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)
	dungeon := make([][]int, height)
	for y := 0; y < height; y++ {
		dungeon[y] = make([]int, width)
		for x := 0; x < width; x++ {
			dungeon[y][x] = TileWall
		}
	}
	numWalkers := 20
	walkLength := 180
	var walkerStartPoints []Point
	for i := 0; i < numWalkers; i++ {
		walkerX := random.Intn(width)
		walkerY := random.Intn(height)
		walkerStartPoints = append(walkerStartPoints, Point{X: walkerX, Y: walkerY})
		for j := 0; j < walkLength; j++ {
			dungeon[walkerY][walkerX] = TileFloor
			direction := random.Intn(4)
			switch direction {
			case 0:
				if walkerY > 1 {
					walkerY--
				}
			case 1:
				if walkerY < height-2 {
					walkerY++
				}
			case 2:
				if walkerX > 1 {
					walkerX--
				}
			case 3:
				if walkerX < width-2 {
					walkerX++
				}
			}
		}
	}
	for i := 1; i < len(walkerStartPoints); i++ {
		p1 := walkerStartPoints[i-1]
		p2 := walkerStartPoints[i]
		carveCorridor(dungeon, p1, p2)
	}
	var floorTiles []Point
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if dungeon[y][x] == TileFloor {
				floorTiles = append(floorTiles, Point{X: x, Y: y})
			}
		}
	}
	var startTile, endTile Point
	if len(floorTiles) > 1 {
		exitIndex := random.Intn(len(floorTiles))
		exitTile := floorTiles[exitIndex]
		dungeon[exitTile.Y][exitTile.X] = TileExit
		endTile = exitTile
		floorTiles = append(floorTiles[:exitIndex], floorTiles[exitIndex+1:]...)
		startIndex := random.Intn(len(floorTiles))
		startTile = floorTiles[startIndex]
		floorTiles = append(floorTiles[:startIndex], floorTiles[startIndex+1:]...)
	}
	numFountains := 3
	for i := 0; i < numFountains && len(floorTiles) > 0; i++ {
		fountainIndex := random.Intn(len(floorTiles))
		fountainTile := floorTiles[fountainIndex]
		dungeon[fountainTile.Y][fountainTile.X] = TileHealth
		floorTiles = append(floorTiles[:fountainIndex], floorTiles[fountainIndex+1:]...)
	}

	itemsToPlace := make(map[Point]string)
	itemsToSpawn := map[string]int{
		"sword": 2,
		"bow":   1,
		"chainmail": 2,
	}
	for itemName, quantity := range itemsToSpawn {
		for i := 0; i < quantity; i++ {
			if len(floorTiles) == 0 {
				break
			}
			idx := random.Intn(len(floorTiles))
			pos := floorTiles[idx]
			itemsToPlace[pos] = itemName
			floorTiles = append(floorTiles[:idx], floorTiles[idx+1:]...)
		}
	}

	return dungeon, floorTiles, startTile, endTile, itemsToPlace
}

func carveCorridor(dungeon [][]int, p1, p2 Point) {
	x1, y1 := p1.X, p1.Y
	x2, y2 := p2.X, p2.Y
	for x := min(x1, x2); x <= max(x1, x2); x++ {
		dungeon[y1][x] = TileFloor
	}
	for y := min(y1, y2); y <= max(y1, y2); y++ {
		dungeon[y][x2] = TileFloor
	}
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}