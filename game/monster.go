package game

import (
	"dunExpo/dungeon"
	"math/rand"
	"time"
)

type MonsterTemplate struct {
	Name      string
	Rune      rune
	Color     string
	HP        int
	Attack    int
	SpawnType string
}

var Bestiary = map[string]MonsterTemplate{
	"goblin": {
		Name:      "Goblin",
		Rune:      'g',
		Color:     dungeon.ColorGreen,
		HP:        5,
		Attack:    1,
		SpawnType: "pack",
	},
	"ogre": {
		Name:      "Ogre",
		Rune:      'O',
		Color:     dungeon.ColorRed,
		HP:        25,
		Attack:    5,
		SpawnType: "single",
	},
	"skeleton_archer": {
		Name:      "Skeleton Archer",
		Rune:      's',
		Color:     dungeon.ColorWhite,
		HP:        10,
		Attack:    2,
		SpawnType: "single",
	},
}

type Monster struct {
	Template  *MonsterTemplate
	Position  dungeon.Point
	CurrentHP int
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
			Template:  &template,
			Position:  spawnPoint,
			CurrentHP: template.HP,
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
					Template:  &template,
					Position:  packMemberSpawnPoint,
					CurrentHP: template.HP,
				}
				monsters = append(monsters, packMonster)
			}
		}
	}

	return monsters
}