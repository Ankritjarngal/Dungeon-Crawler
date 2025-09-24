package game

import (
	"dunExpo/dungeon"
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
		HP:           8,
		Attack:       6,
		SpawnType:    "pack",
		VisionRadius: 8,
		LeashRadius:  12,
	},
	"ogre": {
		Name:         "Ogre",
		Rune:         'O',
		Color:        dungeon.ColorRed,
		HP:           25,
		Attack:       20,
		SpawnType:    "single",
		VisionRadius: 6,
		LeashRadius:  20,
	},
	"skeleton_archer": {
		Name:         "Skeleton Archer",
		Rune:         's',
		Color:        dungeon.ColorWhite,
		HP:           15,
		Attack:       12,
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

	if newPos == state.Player.Position {
		return 
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