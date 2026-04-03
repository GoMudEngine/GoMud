// Raise Vampire — perform the forbidden rite to create a blood-drinking undead noble
// School: manifestation — scales with charisma + manifestation skill
// Mob: Risen Vampire (mob 304)
// Minimum corpse statpool: 300 (only the most formidable remains can survive)

var SUMMON_MOB_ID = 304;
var BASE_STAT_POOL = 100;
var MIN_CORPSE_POOL = 300;
var TYPE_NAME = 'Risen Vampire';

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

    SendUserMessage(sourceActor.UserId(), 'You begin the forbidden rite, weaving blood-dark energy through the remains. This will either work or unmake the corpse entirely...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true) + ' begins a low, rhythmic incantation over the remains. The shadows around them deepen and lean inward.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor, spellAggro) {
    SendUserMessage(sourceActor.UserId(), 'Blood-dark energy pulses through the corpse in rhythmic waves, hunting for the thread of self that the dark gift requires. Something almost stirs...');
}

function onMagic(sourceActor, targetActor, spellAggro) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The rite collapses — the dark gift finds no master to bind it.');
        return true;
    }

    var room = GetRoom(sourceActor.GetRoomId());
    var corpses = room.GetCorpses();
    if (!corpses || corpses.length === 0) {
        SendUserMessage(sourceActor.UserId(), 'The remains have vanished before your spell could take hold.');
        return true;
    }

    var target = null;
    var targetName = (spellAggro && spellAggro.length > 0) ? spellAggro : '';
    for (var i = 0; i < corpses.length; i++) {
        var corpse = corpses[i];
        if (corpse.IsPlayerCorpse()) continue;
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
        SendUserMessage(sourceActor.UserId(), 'The dark gift floods the corpse — and rejects it. The rite collapses with a sound like breaking glass. Something went wrong.');
        return true;
    }

    mob.CharmSet(sourceActor.UserId(), 99999);
    sourceActor.AddCompanion(mob.InstanceId(), 'raised', TYPE_NAME);
    room.RemoveCorpse(target.Index());

    SendUserMessage(sourceActor.UserId(), 'Dark energy flows from your hands into the remains of ' + target.Name() + '. The corpse arches upward, pale flesh flushing to alabaster, eyes snapping open — burning red, aware, and deeply, coldly intelligent. It inclines its head to you. The dark gift has taken.');
    SendRoomMessage(sourceActor.GetRoomId(), 'The remains of ' + target.Name() + ' convulse and then grow very still as ' + sourceActor.GetCharacterName(true) + '\'s dark rite takes hold. A vampire rises — pale, elegant, and deadly — and regards its new master with cold red eyes.', sourceActor.UserId());
    return true;
}
