// Basin Scholar - Mob 79
// Handles quest item delivery for Quest 3 (The Scholar's Collection).
// When a player gives the bone totem or spore sac, auto-advance the quest.

const BONE_TOTEM_ID = 14;
const SPORE_SAC_ID = 40008;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    // Only care about item gives, not gold
    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    var itemId = eventDetails.item.ItemId;

    // Bone totem delivery — advance start -> totem
    if ( itemId == BONE_TOTEM_ID ) {
        if ( user.HasQuest("3-start") && !user.HasQuest("3-totem") ) {
            user.GiveQuest("3-totem");
            mob.Command('say Exquisite. The carving technique is post-mutation -- look at how the finger grooves accommodate clawed digits.', 1.0);
            mob.Command('say Now I need a spore sac from the cave crawlers deeper in the tunnels. It will complete the ecological picture.', 3.0);
            return true;
        }
        // Has the totem but wrong quest state
        mob.Command('say Fascinating, but I already have what I need from the totems. Thank you.');
        return true;
    }

    // Spore sac delivery — advance totem -> end
    if ( itemId == SPORE_SAC_ID ) {
        if ( user.HasQuest("3-totem") && !user.HasQuest("3-end") ) {
            user.GiveQuest("3-end");
            mob.Command('say A spore sac! And intact! The spore density is extraordinary.', 1.0);
            mob.Command('say You have given the guild more data than a decade of field expeditions. I am deeply grateful.', 3.0);
            return true;
        }
        // Has the sac but wrong quest state
        mob.Command('say I appreciate the specimen, but I need the bone totem first to establish context.');
        return true;
    }

    // Not a quest item
    mob.Command('say Interesting, but not what my research requires. Thank you though.');
    return true;
}
