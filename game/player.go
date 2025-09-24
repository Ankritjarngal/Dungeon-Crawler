package game

import "dunExpo/dungeon"

type Player struct {
	ID       string
	Position dungeon.Point
	HP       int
	MaxHP    int
	Attack   int
}

func NewPlayer(id string, startPos dungeon.Point) *Player {
	return &Player{
		ID:       id,
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
		for _, otherPlayer := range state.Players {
			if p.ID != otherPlayer.ID && newPos == otherPlayer.Position {
				return nil
			}
		}
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

func ProcessPlayerCommand(playerID, command string, state *GameState) {
	player, ok := state.Players[playerID]
	if !ok {
		return
	}
	var attackedMonster *Monster
	switch command {
	case "w":
		attackedMonster = player.Move(0, -1, state)
	case "a":
		attackedMonster = player.Move(-1, 0, state)
	case "s":
		attackedMonster = player.Move(0, 1, state)
	case "d":
		attackedMonster = player.Move(1, 0, state)
	}

	if attackedMonster != nil {
		attackedMonster.CurrentHP -= player.Attack
		if attackedMonster.CurrentHP > 0 {
			player.HP -= attackedMonster.Template.Attack
		}
	}
}