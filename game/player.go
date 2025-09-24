package game

import "dunExpo/dungeon"

type Player struct {
	Position dungeon.Point	
	HP int
	MaxHP int
	Attack int
}

func NewPlayer(startPos dungeon.Point) *Player {
	return &Player{
		Position: startPos,
		HP:100,
		MaxHP: 100,
		Attack: 10,
	}
}


func (p *Player) Move(dx, dy int, dungeonMap [][]int) {
	newPos := dungeon.Point{X: p.Position.X + dx, Y: p.Position.Y + dy}

	if newPos.X < 0 || newPos.X >= len(dungeonMap[0]) {
		return
	}
	if newPos.Y < 0 || newPos.Y >= len(dungeonMap) {
		return
	}

	if dungeonMap[newPos.Y][newPos.X] != dungeon.TileWall {
		p.Position = newPos
	}
}