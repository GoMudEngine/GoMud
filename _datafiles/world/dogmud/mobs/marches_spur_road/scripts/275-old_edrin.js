// Old Edrin - Mob 275
// Hermit caster boss. Acts muddled until attacked, then reveals himself
// as a powerful caster and summons elemental guardians.

var revealed = false;

function onHurt(mob, room, eventDetails) {
    if (revealed) { return false; }
    revealed = true;

    // Get the attacker's name so elementals can target them
    var attackerName = '';
    if (eventDetails.sourceId > 0) {
        var attacker = GetUser(eventDetails.sourceId);
        if (attacker) {
            attackerName = attacker.GetCharacterName(false);
        }
    }

    // Drop the act
    mob.Command('say ...ah.');
    mob.Command('emote straightens slowly, the stoop vanishing from his spine. His milky eyes clear to sharp, pale blue. The tremor in his hands stops.');
    mob.Command('say You should not have done that.');

    // Summon three elementals
    room.SendText('Old Edrin raises his staff and speaks three words in a language that predates the road, the fence, and the colony itself.');

    var fire = room.SpawnMob(313);
    var earth = room.SpawnMob(311);
    var water = room.SpawnMob(310);

    room.SendText('The air splits. Fire gathers from nothing. Stone tears itself from the ground. Water condenses from the humidity and takes shape. Three elementals stand beside the old man, burning and grinding and flowing.');

    // Elementals attack whoever hit Edrin
    if (attackerName.length > 0) {
        if (fire) { fire.Command('attack ' + attackerName); }
        if (earth) { earth.Command('attack ' + attackerName); }
        if (water) { water.Command('attack ' + attackerName); }
    }

    mob.Command('emote levels his staff at you, and the air between you and him begins to shimmer with heat.');

    return false;
}

function onIdle(mob, room) {
    if (!revealed) { return false; }
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
