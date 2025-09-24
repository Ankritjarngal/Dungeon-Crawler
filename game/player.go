package game

import (
	"dunExpo/dungeon"
	"fmt"
)

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

func ProcessPlayerCommand(playerID, command string, state *GameState) map[string]bool {
	playersToRemove := make(map[string]bool)
	player, ok := state.Players[playerID]
	if !ok {
		return playersToRemove
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
		damage := player.Attack
		attackedMonster.CurrentHP -= damage
		state.AddMessage(fmt.Sprintf("%s attacks the %s for %d damage!", player.ID, attackedMonster.Template.Name, damage))

		if attackedMonster.CurrentHP > 0 {
			damage = attackedMonster.Template.Attack
			player.HP -= damage
			state.AddMessage(fmt.Sprintf("%s attacks %s for %d damage!", attackedMonster.Template.Name, player.ID, damage))
			if player.HP <= 0 {
				playersToRemove[playerID] = true
				state.AddMessage(fmt.Sprintf("%s has been defeated!", player.ID))
			}
		} else {
			state.AddMessage(fmt.Sprintf("%s is defeated!", attackedMonster.Template.Name))
		}
	}

	var survivingMonsters []*Monster
	for _, m := range state.Monsters {
		if m.CurrentHP > 0 {
			survivingMonsters = append(survivingMonsters, m)
		}
	}
	state.Monsters = survivingMonsters

	if p, ok := state.Players[playerID]; ok {
		if state.Dungeon[p.Position.Y][p.Position.X] == dungeon.TileHealth {
			healAmount := 10
			p.HP += healAmount
			if p.HP > p.MaxHP {
				p.HP = p.MaxHP
			}
			state.Dungeon[p.Position.Y][p.Position.X] = dungeon.TileFloor
			state.AddMessage(fmt.Sprintf("%s drinks from the fountain and recovers %d HP.", p.ID, healAmount))
		}

		if p.Position == state.ExitPos {
			playersToRemove[playerID] = true
			state.AddMessage(fmt.Sprintf("%s has escaped the dungeon! VICTORY!", p.ID))
		}
	}

	return playersToRemove
}