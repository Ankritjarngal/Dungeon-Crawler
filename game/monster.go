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
}

var Bestiary = map[string]MonsterTemplate{
	"goblin": {
		Name:         "Goblin",
		Rune:         'g',
		Color:        dungeon.ColorGreen,
		HP:           10,
		Attack:       5,
		SpawnType:    "pack",
		VisionRadius: 8,
		LeashRadius:  12,
	},
	"ogre": {
		Name:         "Ogre",
		Rune:         'O',
		Color:        dungeon.ColorRed,
		HP:           21,
		Attack:       18,
		SpawnType:    "single",
		VisionRadius: 6,
		LeashRadius:  20,
	},
	"skeleton_archer": {
		Name:         "Skeleton Archer",
		Rune:         's',
		Color:        dungeon.ColorWhite,
		HP:           15,
		Attack:       9,
		SpawnType:    "single",
		VisionRadius: 12,
		LeashRadius:  10,
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
	totalMonstersToSpawn := 15
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
			for j := 0; j < 3; j++ {
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

		distToPlayer := minDist
		distToSpawn := Distance(monster.Position, monster.SpawnPoint)
		visionRadius := monster.Template.VisionRadius
		leashRadius := monster.Template.LeashRadius

		if distToPlayer == 1 {
			damage := monster.Template.Attack
			closestPlayer.HP -= damage
			state.AddMessage(fmt.Sprintf("%s attacks %s for %d damage!", monster.Template.Name, closestPlayer.ID[0:4], damage))
			if closestPlayer.HP <= 0 {
				closestPlayer.Status = "defeated"
				state.AddMessage(fmt.Sprintf("%s has been defeated by a %s!", closestPlayer.ID[0:4], monster.Template.Name))
			}
			continue
		}

		if distToPlayer <= visionRadius && distToSpawn < leashRadius {
			dx, dy := 0, 0
			if closestPlayer.Position.X > monster.Position.X {
				dx = 1
			} else if closestPlayer.Position.X < monster.Position.X {
				dx = -1
			}
			if closestPlayer.Position.Y > monster.Position.Y {
				dy = 1
			} else if closestPlayer.Position.Y < monster.Position.Y {
				dy = -1
			}
			if rand.Intn(2) == 0 {
				monster.Move(dx, 0, state)
			} else {
				monster.Move(0, dy, state)
			}
		} else if distToSpawn > 0 {
			dx, dy := 0, 0
			if monster.SpawnPoint.X > monster.Position.X {
				dx = 1
			} else if monster.SpawnPoint.X < monster.Position.X {
				dx = -1
			}
			if monster.SpawnPoint.Y > monster.Position.Y {
				dy = 1
			} else if monster.SpawnPoint.Y < monster.Position.Y {
				dy = -1
			}
			if rand.Intn(2) == 0 {
				monster.Move(dx, 0, state)
			} else {
				monster.Move(0, dy, state)
			}
		} else {
			direction := rand.Intn(4)
			switch direction {
			case 0:
				monster.Move(0, -1, state)
			case 1:
				monster.Move(0, 1, state)
			case 2:
				monster.Move(-1, 0, state)
			case 3:
				monster.Move(1, 0, state)
			}
		}
	}
}