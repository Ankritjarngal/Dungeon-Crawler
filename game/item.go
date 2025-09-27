package game

import "dunExpo/dungeon"

type Item struct {
	Name      string
	Rune      rune
	Color     string
	IsWeapon  bool
	Damage    int
	Range     int
}

var ItemTemplates = map[string]Item{
	"sword": {
		Name:     "Sword",
		Rune:     '/',
		Color:    dungeon.ColorWhite,
		IsWeapon: true,
		Damage:   15,
		Range:    1,
	},
	"bow": {
		Name:     "Bow",
		Rune:     '(',
		Color:    dungeon.ColorWhite,
		IsWeapon: true,
		Damage:   7,
		Range:    5,
	},
}