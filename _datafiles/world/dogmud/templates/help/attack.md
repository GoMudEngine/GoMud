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

