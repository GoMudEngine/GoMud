// Raise Wraith — tear a spirit from its remains and bind it as an ethereal servant
// School: manifestation — scales with charisma + manifestation skill
// Mob: Bound Wraith (mob 302)
// Minimum corpse statpool: 120 (requires a creature of considerable essence)

var SUMMON_MOB_ID = 302;
var BASE_STAT_POOL = 70;
var MIN_CORPSE_POOL = 120;
var TYPE_NAME = 'Bound Wraith';

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

    SendUserMessage(sourceActor.UserId(), 'You reach beyond flesh and bone, grasping for the shadow-self that still clings to these remains...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true) + ' raises both hands, pulling at something invisible above the remains. The air grows cold and still.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor, spellAggro) {
    SendUserMessage(sourceActor.UserId(), 'Ethereal threads weave through the air, pulling at the lingering spirit. Something cold and furious stirs at the edge of your vision...');
}

function onMagic(sourceActor, targetActor, spellAggro) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The spirit slips free, unmade and unbound.');
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
        SendUserMessage(sourceActor.UserId(), 'The spirit tears free but refuses your binding. It dissolves into cold mist. Something went wrong.');
        return true;
    }

    mob.CharmSet(sourceActor.UserId(), 99999);
    sourceActor.AddCompanion(mob.InstanceId(), 'raised', TYPE_NAME);
    room.RemoveCorpse(target.Index());

    SendUserMessage(sourceActor.UserId(), 'Dark energy flows from your hands into the remains of ' + target.Name() + '. The body collapses to ash as something cold and luminous tears free — a wraith, bound by your will, hovering in silent, furious attendance.');
    SendRoomMessage(sourceActor.GetRoomId(), 'The remains of ' + target.Name() + ' dissolve to ash as ' + sourceActor.GetCharacterName(true) + '\'s power tears something free of them. A cold, glowing wraith drifts into position at their side.', sourceActor.UserId());
    return true;
}
