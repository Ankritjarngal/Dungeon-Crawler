package game

import "dunExpo/dungeon"

type Player struct {
	Position dungeon.Point
	HP       int
	MaxHP    int
	Attack   int
}

func NewPlayer(startPos dungeon.Point) *Player {
	return &Player{
		Position: startPos,
		HP:       100,
		MaxHP:    100,
		Attack:   10,
	}
}

func (p *Player) Move(dx, dy int, state *GameState) *Monster {
	newPos := dungeon.Point{X: p.Position.X + dx, Y: p.Position.Y + dy}

	for _, monster := range state.Monsters {
		if monster.Position == newPos {
			return monster
		}
	}

	if newPos.X < 0 || newPos.X >= dungeon.MapWidth {
		return nil
	}
	if newPos.Y < 0 || newPos.Y >= dungeon.MapHeight {
		return nil
	}
	if state.Dungeon[newPos.Y][newPos.X] != dungeon.TileWall {
		p.Position = newPos
	}

	return nil
}

func Distance(p1, p2 dungeon.Point) int {
	dx := p1.X - p2.X
	if dx < 0 {
		dx = -dx
	}
	dy := p1.Y - p2.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}