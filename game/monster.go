package game

import (
	"dunExpo/dungeon"
	"math/rand"
	"time"
)

func SpawnMonsters(validSpawnPoints []dungeon.Point) []dungeon.Point {
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)

	totalMonsters := 25
	var monsterLocations []dungeon.Point

	for i := 0; i < totalMonsters && len(validSpawnPoints) > 0; i++ {
		randomIndex := random.Intn(len(validSpawnPoints))
		spawnPoint := validSpawnPoints[randomIndex]

		monsterLocations = append(monsterLocations, spawnPoint)

		validSpawnPoints = append(validSpawnPoints[:randomIndex], validSpawnPoints[randomIndex+1:]...)
	}

	return monsterLocations
}
