package game

import "dunExpo/dungeon"

type Player struct {
	Position dungeon.Point	
	HP int
	MaxHP int
	Attack int
}

func NewPlayer(startPos dungeon.Point) *Player {
	return &Player{
		Position: startPos,
		HP:100,
		MaxHP: 100,
		Attack: 10,
	}
}