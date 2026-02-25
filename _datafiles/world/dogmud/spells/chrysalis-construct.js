// Chrysalis Construct — permanent summon spell
// Requires: Chrysalis Core (item 40010) consumed on cast
// Spawns: Chrysalis Construct (mob 110), charmed permanently
// Limit: one construct per caster

var COMPONENT_ITEM_ID = 40010;
var SUMMON_MOB_ID = 110;
var SUMMON_KEY = 'chrysalis-construct-active';

function onCast(sourceActor, targetActor) {
    // Check if caster already has a construct
    var existing = sourceActor.GetMiscCharacterData(SUMMON_KEY);
    if (existing && existing !== '' && existing !== '0') {
        SendUserMessage(sourceActor.UserId(), 'You already have a Chrysalis Construct bound to your will.');
        return false;
    }

    // Check for component
    if (!sourceActor.HasItemId(COMPONENT_ITEM_ID)) {
        SendUserMessage(sourceActor.UserId(), 'You need a Chrysalis Core to shape a construct.');
        return false;
    }

    SendUserMessage(sourceActor.UserId(), 'You hold the Chrysalis Core aloft, pouring your conviction into its pulsing depths...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' holds a pulsing orb aloft, Chrysalis energy swirling around them.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'The Core fractures and reforms, matter coalescing into a towering shape...');
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
        SendUserMessage(sourceActor.UserId(), 'The Chrysalis Core is gone. The spell fizzles.');
        return;
    }

    // Spawn the construct
    var room = GetRoom(sourceActor.GetRoomId());
    var construct = room.SpawnMob(SUMMON_MOB_ID);
    if (!construct) {
        SendUserMessage(sourceActor.UserId(), 'The Chrysalis matter refuses to take form. Something is wrong.');
        return;
    }

    // Charm permanently (99999 rounds ≈ forever)
    construct.CharmSet(sourceActor.UserId(), 99999);

    // Track the summon
    sourceActor.SetMiscCharacterData(SUMMON_KEY, '1');

    SendUserMessage(sourceActor.UserId(), 'The Core shatters and reforms — a hulking Chrysalis Construct rises before you, bound to your will!');
    SendRoomMessage(sourceActor.GetRoomId(), 'Chrysalis matter erupts from a shattered core, coalescing into a massive construct that looms beside '+sourceActor.GetCharacterName(true)+'.', sourceActor.UserId());
}
