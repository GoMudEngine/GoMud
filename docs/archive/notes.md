# TODO

* Create a skill for using magical objects
  * Possibly not use a charge?

* Potion Mixing?
  * INPUT items
  * OUTPUT a unique potion that provides a buff?

* Mutable buff characteristics?
  * Problematic since buffs are determined by ID currently.

* Shapechanging via race changes? Should work... just need to track reverting back.
  * Could be a buff, would need mutable buffs to know what to turn back into, or track this data somehow.

# Basic thoughts on weapons damage guidelines

Dice qty goes up as weapon gets more special
Bonus damage due to enchantment or special quality ("Fine")

- Sticks/Junks etc.          1d2
- 1H Sm (Daggers, claws)     1d4
  - (+Speed)
- 1H Md 
  - Slashing/Stabbing        1d6
  - Cleaving                 1d8
    - (-Speed)
- 2H Md
  - Slashing/Stabbing        1d8
  - Cleaving                 1d8
    - (-Speed)
- 2H Lg
  - Slashing/Stabbing        1d10
    - (-Speed)
  - Cleaving                 1d10
    - (-Speed)

# Weapon speeds?

Currently weapons have a number of rounds extra they must wait between attacks: `waitrounds: 1` for example.

Consider other ways of scaling weapon speed. For example, in order to achieve an extra attack every 2 or 3 rounds.

This would probably require some sort of "energy" pool that gets incremented by round and decremented by attack.
Weapons would need a "speed" or "energy" characteristic that would drive this (possibly a float scaler?).

Example:
`1.0` - default
`0.5` - 1 attack every 2 rounds
`2.0` - 2 attacks every 1 round

What would this mean for weapons that have extra attacks already? Those would be freebies?

# Combat ideas

* When anyone in the party is hated/aggrod, entire party becomes a viable target
  * Includ eother players AND MOBS in the potential target list.

* Consider allowing mobs to change targets mid-fight? Should be transparent though... not another "Gets ready to attack" announcement.

# Mob ideas

* Movement along a pre-scripted path?

* Allow mobs to party/group and move together?
