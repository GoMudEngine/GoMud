// Raise Spectre — compress a powerful soul's shadow into a being of cold malice
// School: manifestation — scales with charisma + manifestation skill
// Mob: Bound Spectre (mob 303)
// Minimum corpse statpool: 200 (requires remains of great potency)

var SUMMON_MOB_ID = 303;
var BASE_STAT_POOL = 90;
var MIN_CORPSE_POOL = 200;
var TYPE_NAME = 'Bound Spectre';

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

    SendUserMessage(sourceActor.UserId(), 'You reach into the deep cold beneath these remains, searching for the heavy, hateful shadow of a powerful soul...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true) + ' crouches over the remains, hands pressing into the ground on either side. A ring of frost spreads outward from the contact.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor, spellAggro) {
    SendUserMessage(sourceActor.UserId(), 'A cold dread emanates from the remains as the soul\'s deepest shadow is compressed and shaped. The air tastes of iron and old grief...');
}

function onMagic(sourceActor, targetActor, spellAggro) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The spectre\'s form shatters before it can coalesce.');
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
        SendUserMessage(sourceActor.UserId(), 'The shadow compresses into almost-form, then collapses inward. The binding failed. Something went wrong.');
        return true;
    }

    mob.CharmSet(sourceActor.UserId(), 99999);
    sourceActor.AddCompanion(mob.InstanceId(), 'raised', TYPE_NAME);
    room.RemoveCorpse(target.Index());

    SendUserMessage(sourceActor.UserId(), 'Dark energy flows from your hands into the remains of ' + target.Name() + '. Everything freezes for a heartbeat — then the shadow tears free, compressed into something dense and hateful. A spectre regards you from within that darkness, bound by your will alone.');
    SendRoomMessage(sourceActor.GetRoomId(), 'The remains of ' + target.Name() + ' go still as death as ' + sourceActor.GetCharacterName(true) + '\'s power rips something cold and dark from within them. A spectre coalesces beside them, radiating icy malice.', sourceActor.UserId());
    return true;
}
