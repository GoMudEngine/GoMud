// Summon Hive Swarm — permanent summon spell
// Requires: Hive Fragment (item 40011) consumed on cast
// Spawns: Hive Swarm (mob 111), charmed permanently
// Limit: one swarm per caster

var COMPONENT_ITEM_ID = 40011;
var SUMMON_MOB_ID = 111;
var SUMMON_KEY = 'hive-swarm-active';

function onCast(sourceActor, targetActor) {
    // Check if caster already has a swarm
    var existing = sourceActor.GetMiscCharacterData(SUMMON_KEY);
    if (existing && existing !== '' && existing !== '0') {
        SendUserMessage(sourceActor.UserId(), 'You already have a Hive Swarm bound to your will.');
        return false;
    }

    // Check for component
    if (!sourceActor.HasItemId(COMPONENT_ITEM_ID)) {
        SendUserMessage(sourceActor.UserId(), 'You need a Hive Fragment to call forth the swarm.');
        return false;
    }

    SendUserMessage(sourceActor.UserId(), 'You crush the Hive Fragment in your fist, feeling thousands of tiny lives stir within...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' crushes a chitinous fragment, a faint buzzing filling the air.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'The buzzing intensifies as countless tiny organisms pour from the shattered fragment...');
}

function onMagic(sourceActor, targetActor) {
    // Consume the component
    var items = sourceActor.GetBackpackItems();
    var consumed = false;
    for (var i = 0; i < items.length; i++) {
        if (items[i].ItemId() === COMPONENT_ITEM_ID) {
            sourceActor.TakeItem(items[i]);
            consumed = true;
            break;
        }
    }
    if (!consumed) {
        SendUserMessage(sourceActor.UserId(), 'The Hive Fragment is gone. The spell fizzles.');
        return;
    }

    // Spawn the swarm
    var room = GetRoom(sourceActor.GetRoomId());
    var swarm = room.SpawnMob(SUMMON_MOB_ID);
    if (!swarm) {
        SendUserMessage(sourceActor.UserId(), 'The tiny organisms scatter and dissolve. Something is wrong.');
        return;
    }

    // Charm permanently (99999 rounds ≈ forever)
    swarm.CharmSet(sourceActor.UserId(), 99999);

    // Track the summon
    sourceActor.SetMiscCharacterData(SUMMON_KEY, '1');

    SendUserMessage(sourceActor.UserId(), 'The fragment erupts into a roiling cloud of iridescent organisms — a Hive Swarm coalesces around you, awaiting your command!');
    SendRoomMessage(sourceActor.GetRoomId(), 'A chitinous fragment shatters in '+sourceActor.GetCharacterName(true)+"'s hand, releasing a dense swarm of tiny Chrysalis creatures that swirl into formation.", sourceActor.UserId());
}
