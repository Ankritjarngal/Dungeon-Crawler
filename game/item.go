package game

import "dunExpo/dungeon"

type Item struct {
	Name       string
	Rune       rune
	Color      string
	IsWeapon   bool
	IsArmor    bool
	Damage     int
	Range      int
	Durability int
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
		Range:    8,
	},
	"chainmail": {
		Name:       "Chainmail",
		Rune:       '#',
		Color:      dungeon.ColorWhite,
		IsArmor:    true,
		Durability: 20,
	},
}