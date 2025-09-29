package game

import (
	"dunExpo/dungeon"
	"fmt"
	"math/rand"
	"time"
)

type MonsterTemplate struct {
	Name         string
	Rune         rune
	Color        string
	HP           int
	Attack       int
	SpawnType    string
	VisionRadius int
	LeashRadius  int
	AttackRange  int
	MovingSpeed  int 
}

var Bestiary = map[string]MonsterTemplate{
	"goblin": {
		Name:         "Goblin",
		Rune:         'g',
		Color:        dungeon.ColorGreen,
		HP:           7,
		Attack:       3,
		SpawnType:    "pack",
		VisionRadius: 8,
		LeashRadius:  12,
		AttackRange:  1,
		MovingSpeed:  2,
	},
	"ogre": {
		Name:         "Ogre",
		Rune:         'O',
		Color:        dungeon.ColorRed,
		HP:           25,
		Attack:       8,
		SpawnType:    "single",
		VisionRadius: 6,
		LeashRadius:  20,
		AttackRange:  1,
		MovingSpeed: 1,
	},
	"skeleton_archer": {
		Name:         "Skeleton Archer",
		Rune:         's',
		Color:        dungeon.ColorWhite,
		HP:           15,
		Attack:       6,
		SpawnType:    "single",
		VisionRadius: 12,
		LeashRadius:  10,
		AttackRange:  6,
		MovingSpeed: 1,
	},
	"bat":{
		Name:         "Bat",
		Rune:         'b',
		Color:        dungeon.ColorMagenta,
		HP:           3,
		Attack:       2,
		SpawnType:    "single",
		VisionRadius: 5,
		LeashRadius:  8,
		AttackRange:  1,
		MovingSpeed: 3,
	},
	
}

type Monster struct {
	Template   *MonsterTemplate
	Position   dungeon.Point
	CurrentHP  int
	SpawnPoint dungeon.Point
}

func (m *Monster) Move(dx, dy int, state *GameState) {
	newPos := dungeon.Point{X: m.Position.X + dx, Y: m.Position.Y + dy}
	if newPos.X < 0 || newPos.X >= dungeon.MapWidth {
		return
	}
	if newPos.Y < 0 || newPos.Y >= dungeon.MapHeight {
		return
	}
	if state.Dungeon[newPos.Y][newPos.X] == dungeon.TileWall {
		return
	}
	if newPos == state.ExitPos {
		return
	}
	for _, p := range state.Players {
		if newPos == p.Position {
			return
		}
	}
	for _, otherMonster := range state.Monsters {
		if m != otherMonster && newPos == otherMonster.Position {
			return
		}
	}
	m.Position = newPos
}

func SpawnMonsters(validSpawnPoints []dungeon.Point) []*Monster {
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)
	totalMonstersToSpawn := 20
	var monsters []*Monster
	var monsterKeys []string
	for k := range Bestiary {
		monsterKeys = append(monsterKeys, k)
	}
	for i := 0; i < totalMonstersToSpawn && len(validSpawnPoints) > 0; i++ {
		randomKey := monsterKeys[random.Intn(len(monsterKeys))]
		template := Bestiary[randomKey]
		randomIndex := random.Intn(len(validSpawnPoints))
		spawnPoint := validSpawnPoints[randomIndex]
		validSpawnPoints = append(validSpawnPoints[:randomIndex], validSpawnPoints[randomIndex+1:]...)
		newMonster := &Monster{
			Template:   &template,
			Position:   spawnPoint,
			CurrentHP:  template.HP,
			SpawnPoint: spawnPoint,
		}
		monsters = append(monsters, newMonster)
		if template.SpawnType == "pack" && len(validSpawnPoints) > 2 {
			for j := 0; j < 2; j++ {
				if len(validSpawnPoints) == 0 {
					break
				}
				packMemberIndex := random.Intn(len(validSpawnPoints))
				packMemberSpawnPoint := validSpawnPoints[packMemberIndex]
				validSpawnPoints = append(validSpawnPoints[:packMemberIndex], validSpawnPoints[packMemberIndex+1:]...)
				packMonster := &Monster{
					Template:   &template,
					Position:   packMemberSpawnPoint,
					CurrentHP:  template.HP,
					SpawnPoint: packMemberSpawnPoint,
				}
				monsters = append(monsters, packMonster)
			}
		}
	}
	return monsters
}
func UpdateMonsters(state *GameState) {
	state.Log = []string{}

	for _, monster := range state.Monsters {
		var closestPlayer *Player
		minDist := -1

		// Find the closest player
		for _, player := range state.Players {
			if player.Status != "playing" {
				continue
			}
			dist := Distance(monster.Position, player.Position)
			if minDist == -1 || dist < minDist {
				minDist = dist
				closestPlayer = player
			}
		}

		if closestPlayer == nil {
			continue
		}

		// Skeleton Archer ranged attack
		if monster.Template.Name == "Skeleton Archer" {
			distToPlayer := Distance(monster.Position, closestPlayer.Position)
			if distToPlayer <= monster.Template.AttackRange &&
				LineOfSight(monster.Position, closestPlayer.Position, state.Dungeon) {

				damage := monster.Template.Attack
				if closestPlayer.EquippedArmor != nil {
					brokenArmor := closestPlayer.EquippedArmor
					closestPlayer.EquippedArmor.Durability -= damage
					state.AddMessage(fmt.Sprintf("%s's armor absorbs %d damage!", closestPlayer.ID[0:4], damage))
					if closestPlayer.EquippedArmor.Durability <= 0 {
						state.AddMessage(fmt.Sprintf("%s's %s breaks!", closestPlayer.ID[0:4], closestPlayer.EquippedArmor.Name))
						closestPlayer.EquippedArmor = nil
						var newInventory []*Item
						for _, item := range closestPlayer.Inventory {
							if item != brokenArmor {
								newInventory = append(newInventory, item)
							}
						}
						closestPlayer.Inventory = newInventory
					}
				} else {
					closestPlayer.HP -= damage
					state.AddMessage(fmt.Sprintf("%s fires an arrow at %s for %d damage!", monster.Template.Name, closestPlayer.ID[0:4], damage))
				}
				if closestPlayer.HP <= 0 {
					closestPlayer.Status = "defeated"
					state.AddMessage(fmt.Sprintf("%s has been defeated by a %s!", closestPlayer.ID[0:4], monster.Template.Name))
				}
				continue
			}
		}

		// Melee attack if adjacent
		if Distance(monster.Position, closestPlayer.Position) == 1 {
			damage := monster.Template.Attack
			if closestPlayer.EquippedArmor != nil {
				brokenArmor := closestPlayer.EquippedArmor
				closestPlayer.EquippedArmor.Durability -= damage
				state.AddMessage(fmt.Sprintf("%s's armor absorbs %d damage!", closestPlayer.ID[0:4], damage))
				if closestPlayer.EquippedArmor.Durability <= 0 {
					state.AddMessage(fmt.Sprintf("%s's %s breaks!", closestPlayer.ID[0:4], closestPlayer.EquippedArmor.Name))
					closestPlayer.EquippedArmor = nil
					var newInventory []*Item
					for _, item := range closestPlayer.Inventory {
						if item != brokenArmor {
							newInventory = append(newInventory, item)
						}
					}
					closestPlayer.Inventory = newInventory
				}
			} else {
				closestPlayer.HP -= damage
				state.AddMessage(fmt.Sprintf("%s attacks %s for %d damage!", monster.Template.Name, closestPlayer.ID[0:4], damage))
			}
			if closestPlayer.HP <= 0 {
				closestPlayer.Status = "defeated"
				state.AddMessage(fmt.Sprintf("%s has been defeated by a %s!", closestPlayer.ID[0:4], monster.Template.Name))
			}
			continue
		}

		// Determine target position: chase player if in vision, else return to spawn
		target := monster.SpawnPoint
		if Distance(monster.Position, closestPlayer.Position) <= monster.Template.VisionRadius &&
			Distance(monster.Position, monster.SpawnPoint) < monster.Template.LeashRadius {
			target = closestPlayer.Position
		}

		// Move towards target according to MovingSpeed
		for step := 0; step < monster.Template.MovingSpeed; step++ {
			dx, dy := 0, 0
			if target.X > monster.Position.X {
				dx = 1
			} else if target.X < monster.Position.X {
				dx = -1
			}
			if target.Y > monster.Position.Y {
				dy = 1
			} else if target.Y < monster.Position.Y {
				dy = -1
			}

			// Randomize X/Y movement like before
			if dx != 0 && dy != 0 {
				if rand.Intn(2) == 0 {
					dy = 0
				} else {
					dx = 0
				}
			}

			if dx != 0 || dy != 0 {
				monster.Move(dx, dy, state)
			} else {
				break
			}
		}
	}
}
