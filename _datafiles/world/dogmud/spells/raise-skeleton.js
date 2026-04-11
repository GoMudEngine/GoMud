// Raise Skeleton — animate bones from a nearby corpse
// School: manifestation — scales with charisma + manifestation skill
// Mob: Skeleton Warrior (mob 300)
// Minimum corpse statpool: 30 (almost any corpse qualifies)

var SUMMON_MOB_ID = 300;
var BASE_STAT_POOL = 60;
var MIN_CORPSE_POOL = 30;
var TYPE_NAME = 'Skeleton Warrior';

var FAIL_TEXTS = [
    'The remains shudder and twitch, but the spark of undeath finds nothing to cling to.',
    'Dark energy seeps into the corpse... and leaks out through a thousand tiny fractures.',
    'Bones rattle briefly, then collapse into dust. There wasn\'t enough left to work with.',
    'The corpse stirs, a hollow mockery of life flickers — then fades. Not enough essence remains.',
    'Your magic courses through the remains, but finds only emptiness.'
];

function onCast(sourceActor, targetActor, spellAggro) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. Release one before raising another.');
        return false;
    }

    var room = GetRoom(sourceActor.GetRoomId());
    var corpses = room.GetCorpses();
    if (!corpses || corpses.length === 0) {
        SendUserMessage(sourceActor.UserId(), 'There are no remains here to work with.');
        return false;
    }

    return true;
}

function onMagic(sourceActor, targetActor, spellAggro) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The bones rattle but do not rise.');
        return true;
    }

    var room = GetRoom(sourceActor.GetRoomId());
    var corpses = room.GetCorpses();
    if (!corpses || corpses.length === 0) {
        SendUserMessage(sourceActor.UserId(), 'The remains have vanished before your spell could take hold.');
        return true;
    }

    // Find target corpse — prefer named match, otherwise first non-player corpse
    var target = null;
    var targetName = (spellAggro && spellAggro.length > 0) ? spellAggro : '';
    for (var i = 0; i < corpses.length; i++) {
        var corpse = corpses[i];
        if (corpse.IsPlayerCorpse()) continue;
        if (corpse.WasCompanion()) continue;
        if (targetName.length > 0 && corpse.Name().toLowerCase().indexOf(targetName.toLowerCase()) === -1) continue;
        target = corpse;
        break;
    }

    if (!target) {
        if (targetName.length > 0) {
            SendUserMessage(sourceActor.UserId(), 'You cannot find a corpse matching that name here.');
        } else {
            SendUserMessage(sourceActor.UserId(), 'There are no suitable remains to animate.');
        }
        return true;
    }

    var corpsePool = target.GetStatTrainingTotal();
    if (corpsePool < MIN_CORPSE_POOL) {
        SendUserMessage(sourceActor.UserId(), FAIL_TEXTS[Math.floor(Math.random() * FAIL_TEXTS.length)]);
        return true;
    }

    var charisma = sourceActor.GetStat('charisma');
    var manifestSkill = sourceActor.GetSkillLevel('manifestation');
    var companionScale = Math.round(BASE_STAT_POOL * (1.0 + charisma / 500.0 + manifestSkill * 0.02));
    var raisedPool = Math.round(companionScale * 0.5 + corpsePool * 0.5);

    var mob = room.SpawnMobScaled(SUMMON_MOB_ID, raisedPool);
    if (!mob) {
        SendUserMessage(sourceActor.UserId(), 'The bones stir and almost rise — then the magic collapses. Something went wrong.');
        return true;
    }

    mob.CharmSet(sourceActor.UserId(), 99999);
    sourceActor.AddCompanion(mob.InstanceId(), 'raised', TYPE_NAME);
    room.RemoveCorpse(target.Index());

    SendUserMessage(sourceActor.UserId(), 'Dark energy flows from your hands into the remains of ' + target.Name() + '. Bones snap upright, stripped clean and gleaming, assembling into something that regards you with empty sockets. It has risen to serve.');
    SendRoomMessage(sourceActor.GetRoomId(), 'The bones of ' + target.Name() + ' snap and clatter upright as ' + sourceActor.GetCharacterName(true) + '\'s dark power floods through them. A skeleton warrior rises, rattling, and takes its place at their side.', sourceActor.UserId());
    return true;
}
