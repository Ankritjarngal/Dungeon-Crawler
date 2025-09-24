package game

import "dunExpo/dungeon"

type GameState struct {
	Dungeon  [][]int
	Monsters []*Monster
	Player   *Player
	ExitPos  dungeon.Point
	Log []string
}