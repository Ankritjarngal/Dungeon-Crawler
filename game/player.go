package game

import (
	"dunExpo/dungeon"
	"fmt"
)

type Player struct {
	ID             string
	Position       dungeon.Point
	HP             int
	MaxHP          int
	Attack         int
	Status         string
	Inventory      []*Item
	EquippedWeapon *Item
	EquippedArmor  *Item
	Target         *dungeon.Point
	VisionRadius   int
}

func NewPlayer(id string, startPos dungeon.Point) *Player {
	return &Player{
		ID:             id,
		Position:       startPos,
		HP:             100,
		MaxHP:          100,
		Attack:         10,
		Status:         "playing",
		Inventory:      []*Item{},
		EquippedWeapon: nil,
		EquippedArmor:  nil,
		VisionRadius:   6,
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

func CalculateVisibility(center dungeon.Point, radius int) map[dungeon.Point]bool {
	visible := make(map[dungeon.Point]bool)
	for y := center.Y - radius; y <= center.Y + radius; y++ {
		for x := center.X - radius; x <= center.X + radius; x++ {
			if x < 0 || x >= dungeon.MapWidth || y < 0 || y >= dungeon.MapHeight {
				continue
			}
			if Distance(center, dungeon.Point{X: x, Y: y}) <= radius {
				visible[dungeon.Point{X: x, Y: y}] = true
			}
		}
	}
	return visible
}

func ProcessPlayerCommand(playerID, command string, state *GameState) (map[string]bool, bool) {
	playersToRemove := make(map[string]bool)
	player, ok := state.Players[playerID]
	if !ok || (player.Status != "playing" && player.Status != "targeting") {
		return playersToRemove, false
	}
	if player.Status == "targeting" {
		switch command {
		case "f":
			if target, ok := FindMonsterAt(state, player.Target); ok {
				damage := player.EquippedWeapon.Damage
				target.CurrentHP -= damage
				state.AddMessage(fmt.Sprintf("%s fires an arrow at the %s for %d damage!", player.ID[0:4], target.Template.Name, damage))
				if target.CurrentHP <= 0 {
					state.AddMessage(fmt.Sprintf("%s is defeated!", target.Template.Name))
				}
			}
			player.Status = "playing"
			player.Target = nil
			return playersToRemove, false
		default:
			player.Status = "playing"
			player.Target = nil
			state.AddMessage("Targeting cancelled.")
			return playersToRemove, true
		}
	}

	var attackedMonster *Monster
	var dx, dy int
	moved := false
	switch command {
	case "w":
		dx, dy, moved = 0, -1, true
	case "a":
		dx, dy, moved = -1, 0, true
	case "s":
		dx, dy, moved = 0, 1, true
	case "d":
		dx, dy, moved = 1, 0, true
	case "g":
		if itemOnGround, ok := state.ItemsOnGround[player.Position]; ok {
			hasDuplicate := false
			for _, existingItem := range player.Inventory {
				if existingItem.Name == itemOnGround.Name {
					hasDuplicate = true
					break
				}
			}
			if hasDuplicate {
				state.AddMessage(fmt.Sprintf("You already have a %s.", itemOnGround.Name))
				return playersToRemove, false
			}
			player.Inventory = append(player.Inventory, itemOnGround)
			if itemOnGround.IsWeapon && player.EquippedWeapon == nil {
				player.EquippedWeapon = itemOnGround
			}
			if itemOnGround.IsArmor && player.EquippedArmor == nil {
				player.EquippedArmor = itemOnGround
			}
			delete(state.ItemsOnGround, player.Position)
			state.AddMessage(fmt.Sprintf("%s picks up the %s.", player.ID[0:4], itemOnGround.Name))
		}
	case "e":
		var weaponsInInventory []*Item
		for _, item := range player.Inventory {
			if item.IsWeapon {
				weaponsInInventory = append(weaponsInInventory, item)
			}
		}
		if len(weaponsInInventory) > 0 {
			if player.EquippedWeapon == nil {
				player.EquippedWeapon = weaponsInInventory[0]
			} else {
				currentIndex := -1
				for i, w := range weaponsInInventory {
					if w == player.EquippedWeapon {
						currentIndex = i
						break
					}
				}
				nextIndex := (currentIndex + 1) % len(weaponsInInventory)
				player.EquippedWeapon = weaponsInInventory[nextIndex]
			}
			state.AddMessage(fmt.Sprintf("%s equips the %s.", player.ID[0:4], player.EquippedWeapon.Name))
		} else {
			state.AddMessage("No weapons in inventory to equip.")
		}
	case "f":
		if player.EquippedWeapon != nil && player.EquippedWeapon.Name == "Bow" {
			target := FindClosestVisibleMonster(state, player)
			if target != nil {
				player.Status = "targeting"
				targetPos := target.Position
				player.Target = &targetPos
				state.AddMessage("Aiming... Press 'f' to fire or any other key to cancel.")
				return playersToRemove, true
			} else {
				state.AddMessage("No valid targets in a straight line.")
			}
		} else {
			state.AddMessage("You don't have a bow equipped!")
		}
	case "D":
		if player.EquippedWeapon != nil {
			if _, ok := state.ItemsOnGround[player.Position]; !ok {
				droppedItem := player.EquippedWeapon
				state.ItemsOnGround[player.Position] = droppedItem
				player.EquippedWeapon = nil
				var newInventory []*Item
				found := false
				for _, item := range player.Inventory {
					if !found && item == droppedItem {
						found = true
						continue
					}
					newInventory = append(newInventory, item)
				}
				player.Inventory = newInventory
				state.AddMessage(fmt.Sprintf("%s drops the %s.", player.ID[0:4], droppedItem.Name))
			} else {
				state.AddMessage("You can't drop an item here.")
			}
		} else {
			state.AddMessage("You have nothing equipped to drop.")
		}
	}

	if moved {
		attackedMonster = player.Move(dx, dy, state)
	}

	if attackedMonster != nil {
		damage := player.Attack
		if player.EquippedWeapon != nil {
			damage = player.EquippedWeapon.Damage
		}
		attackedMonster.CurrentHP -= damage
		state.AddMessage(fmt.Sprintf("%s attacks the %s for %d damage!", player.ID[0:4], attackedMonster.Template.Name, damage))
		if attackedMonster.CurrentHP > 0 {
			damage = attackedMonster.Template.Attack
			if player.EquippedArmor != nil {
				brokenArmor := player.EquippedArmor
				player.EquippedArmor.Durability -= damage
				state.AddMessage(fmt.Sprintf("%s's armor absorbs %d damage!", player.ID[0:4], damage))
				if player.EquippedArmor.Durability <= 0 {
					state.AddMessage(fmt.Sprintf("%s's %s breaks!", player.ID[0:4], player.EquippedArmor.Name))
					player.EquippedArmor = nil
					var newInventory []*Item
					for _, item := range player.Inventory {
						if item != brokenArmor {
							newInventory = append(newInventory, item)
						}
					}
					player.Inventory = newInventory
				}
			} else {
				player.HP -= damage
				state.AddMessage(fmt.Sprintf("%s attacks %s for %d damage!", attackedMonster.Template.Name, player.ID[0:4], damage))
			}
			if player.HP <= 0 {
				player.Status = "defeated"
				state.AddMessage(fmt.Sprintf("%s has been defeated!", player.ID[0:4]))
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

	if p, ok := state.Players[playerID]; ok && p.Status == "playing" {
		if state.Dungeon[p.Position.Y][p.Position.X] == dungeon.TileHealth {
			healAmount := 10
			p.HP += healAmount
			if p.HP > p.MaxHP {
				p.HP = p.MaxHP
			}
			state.Dungeon[p.Position.Y][p.Position.X] = dungeon.TileFloor
			state.AddMessage(fmt.Sprintf("%s drinks from the fountain.", p.ID[0:4]))
		}
		if p.Position == state.ExitPos {
			state.AddMessage(fmt.Sprintf("%s has reached the exit! The party is victorious!", p.ID[0:4]))
			for id := range state.Players {
				playersToRemove[id] = true
			}
			return playersToRemove, false
		}
		distToExit := Distance(p.Position, state.ExitPos)
		if distToExit <= 2 {
			state.AddMessage("You see the exit shimmering nearby.")
		} else if distToExit <= 5 {
			state.AddMessage("You feel a draft from a nearby exit.")
		}
	}
	return playersToRemove, false
}

func GetLineOfSightPath(p1, p2 dungeon.Point) []dungeon.Point {
	var path []dungeon.Point
	x1, y1 := p1.X, p1.Y
	x2, y2 := p2.X, p2.Y
	dx := x2 - x1
	if dx < 0 {
		dx = -dx
	}
	dy := y2 - y1
	if dy < 0 {
		dy = -dy
	}
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy
	for {
		path = append(path, dungeon.Point{X: x1, Y: y1})
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
	return path
}

func LineOfSight(p1, p2 dungeon.Point, dungeonMap [][]int) bool {
	path := GetStraightLinePath(p1, p2)
	if len(path) == 0 {
		return false
	}
	for i, p := range path {
		if i == 0 || i == len(path)-1 {
			continue
		}
		if dungeonMap[p.Y][p.X] == dungeon.TileWall {
			return false
		}
	}
	return true
}

func FindMonsterAt(state *GameState, pos *dungeon.Point) (*Monster, bool) {
	if pos == nil {
		return nil, false
	}
	for _, m := range state.Monsters {
		if m.Position == *pos {
			return m, true
		}
	}
	return nil, false
}

func FindClosestVisibleMonster(state *GameState, p *Player) *Monster {
	var target *Monster
	minDist := -1
	for _, m := range state.Monsters {
		if m.CurrentHP <= 0 {
			continue
		}
		if m.Position.X != p.Position.X && m.Position.Y != p.Position.Y {
			continue
		}
		dist := Distance(p.Position, m.Position)
		if (minDist == -1 || dist < minDist) && dist <= p.EquippedWeapon.Range {
			if LineOfSight(p.Position, m.Position, state.Dungeon) {
				minDist = dist
				target = m
			}
		}
	}
	return target
}

func GetStraightLinePath(p1, p2 dungeon.Point) []dungeon.Point {
	var path []dungeon.Point
	x1, y1 := p1.X, p1.Y
	x2, y2 := p2.X, p2.Y
	if x1 == x2 {
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		for y := y1; y <= y2; y++ {
			path = append(path, dungeon.Point{X: x1, Y: y})
		}
	} else if y1 == y2 {
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		for x := x1; x <= x2; x++ {
			path = append(path, dungeon.Point{X: x, Y: y1})
		}
	}
	return path
}