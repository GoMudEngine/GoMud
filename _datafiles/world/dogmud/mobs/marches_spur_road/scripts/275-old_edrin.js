// Old Edrin - Mob 275
// Hermit caster boss. Acts muddled until attacked, then reveals himself
// as a powerful caster and summons hostile elemental guardians.
// Anti-cheese: elementals are SetHostile (permanent aggro), Edrin
// re-aggros via onIdle. Fleeing gains nothing.

var revealed = false;
var targetName = '';

function onHurt(mob, room, eventDetails) {
    // Remember who started this
    if (eventDetails.sourceId > 0 && targetName === '') {
        var attacker = GetUser(eventDetails.sourceId);
        if (attacker) {
            targetName = attacker.GetCharacterName(false);
        }
    }

    if (revealed) { return false; }
    revealed = true;

    // Drop the act
    mob.Command('say ...ah.');
    mob.Command('emote straightens slowly, the stoop vanishing from his spine. His milky eyes clear to sharp, pale blue. The tremor in his hands stops.');
    mob.Command('say You should not have done that.');

    // Summon three hostile elementals
    room.SendText('Old Edrin raises his staff and speaks three words in a language that predates the road, the fence, and the colony itself.');

    var fire = room.SpawnMob(313);
    var earth = room.SpawnMob(311);
    var water = room.SpawnMob(310);

    if (fire) { fire.SetHostile(true); fire.Command('attack ' + targetName); }
    if (earth) { earth.SetHostile(true); earth.Command('attack ' + targetName); }
    if (water) { water.SetHostile(true); water.Command('attack ' + targetName); }

    room.SendText('The air splits. Fire gathers from nothing. Stone tears itself from the ground. Water condenses from the humidity and takes shape. Three elementals stand beside the old man, burning and grinding and flowing.');

    mob.Command('emote levels his staff at you, and the air between you and him begins to shimmer with heat.');

    return false;
}

function onIdle(mob, room) {
    if (!revealed) { return false; }

    // Re-aggro if not fighting and players present
    if (!mob.IsAggroed()) {
        var players = room.GetPlayers();
        if (players && players.length > 0) {
            if (targetName.length > 0) {
                mob.Command('attack ' + targetName);
            }
        }
    }

    return false;
}

function onDie(mob, room, eventDetails) {
    room.SendText('Old Edrin sinks to one knee, staff cracking beneath him. The elementals shudder and dissolve -- fire guttering, stone crumbling, water splashing to nothing. The old man looks up with those clear, pale eyes.');
    room.SendText('"Hm," he says. "Stronger than you looked." And then he is still.');

    // Unlock the back room
    room.SetLocked('west', false);
    room.SendText('The grass curtain over the back doorway falls away, the ward holding it shut broken with its maker.');

    // Spawn loot in the back room (4037)
    var backRoom = GetRoom(4037);
    if (backRoom) {
        if (Math.random() < 0.75) { backRoom.SpawnItem(40010); }
        if (Math.random() < 0.75) { backRoom.SpawnItem(40027); }
        if (Math.random() < 0.75) { backRoom.SpawnItem(40004); }
        if (Math.random() < 0.75) { backRoom.SpawnItem(40009); }
    }

    return true;
}
