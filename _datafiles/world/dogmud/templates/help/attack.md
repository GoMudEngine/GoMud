# Help for ~attack~

The ~attack~ command engages in combat with a player or NPC.  
Once started, combat continues until someone flees or someone dies.

## Usage:

  ~attack goblin~  
  This would start combat with the goblin.

*Chance to Hit* is calculated as follows:

- **attackersSpeed** / (**atackerSpeed** + **defenderSpeed**) * **70** + **30**

You always have a minimum 5% chance to miss, and a minimum 5% chance to hit.

*Crit Chance* is calculated as follows: 

- (**Strength** + **Speed**) / (**attackerLevel** - **defenderLevel**) + **5**

## Weapon Reach in Grapples

When you are **grappling** (in a clinch or on the ground), the
*reach* of your wielded weapon matters. Long weapons (greatswords,
spears, polearms, staves) can't be swung freely and deal reduced
damage as pommel or hilt strikes. Short weapons (daggers, fists,
claws, wands) stay fully effective. A bladed weapon used as a
bludgeon will narrate as such — *bash* or *smash* rather than
*slash* or *stab*.

Carrying a dagger as your offhand is a sound counter to grapplers:
each swing is evaluated independently, so the dagger keeps cutting
while your main-hand sword is hampered.

See also: ~help reach~, ~help grapple~, ~help weapon-combat~

