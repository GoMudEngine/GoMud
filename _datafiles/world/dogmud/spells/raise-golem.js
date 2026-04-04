// Raise Flesh Golem — stitch multiple corpses into a massive animate construct
// School: manifestation — scales with charisma + manifestation skill
// Mob: Flesh Golem (mob 305)
// Minimum corpse statpool: 500 (only the most potent remains suffice)

var SUMMON_MOB_ID = 305;
var BASE_STAT_POOL = 120;
var MIN_CORPSE_POOL = 500;
var TYPE_NAME = 'Flesh Golem';

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

    SendUserMessage(sourceActor.UserId(), 'You begin the brutal stitching-work — not resurrection but construction, forcing flesh and sinew to knit according to your will...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true) + ' gestures over the remains with sweeping, surgical motions. The ground shudders faintly with each pass of their hands.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor, spellAggro) {
    SendUserMessage(sourceActor.UserId(), 'The ground trembles as flesh begins to knit and bulk, assembling into something vast and purposeful. The air reeks of raw meat and ozone...');
}

function onMagic(sourceActor, targetActor, spellAggro) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The half-formed golem collapses back into inert flesh.');
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
        SendUserMessage(sourceActor.UserId(), 'The construct reaches critical mass and then tears itself apart. The animating will found no foothold. Something went wrong.');
        return true;
    }

    mob.CharmSet(sourceActor.UserId(), 99999);
    sourceActor.AddCompanion(mob.InstanceId(), 'raised', TYPE_NAME);
    room.RemoveCorpse(target.Index());

    SendUserMessage(sourceActor.UserId(), 'Dark energy flows from your hands into the remains of ' + target.Name() + '. The flesh heaves and swells, stitching itself into a massive, lurching form that blocks the light when it stands. It turns toward you, awaiting direction. The golem lives — after a fashion.');
    SendRoomMessage(sourceActor.GetRoomId(), 'The remains of ' + target.Name() + ' writhe and expand as ' + sourceActor.GetCharacterName(true) + '\'s necromantic surgery takes hold. A massive flesh golem hauls itself upright with a sound like tearing canvas and turns its blank face toward its new master.', sourceActor.UserId());
    return true;
}
