// Geomancer Rhett - Mob 242
// Handles quest item delivery for Quest 11 (The Windwarden's Dilemma).
// When a player gives the windstone sample (item 40032), auto-advance.

const SAMPLE_ITEM_ID = 40032;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    // Windstone sample delivery for Q11
    if ( eventDetails.item.ItemId == SAMPLE_ITEM_ID ) {
        if ( user.HasQuest("11-rhett") && !user.HasQuest("11-end") ) {
            mob.Command('say Look at that lattice structure!', 1.0);
            mob.Command('say The resonance potential is even higher than my estimates. These are perfect.', 2.5);
            mob.Command('say Ask me about further work when you are ready. I have plans for this crystal.', 4.0);
            user.GiveQuest("11-end");
            return true;
        }
        mob.Command('say I already have what I need. Thank you though!');
        return true;
    }

    // Not the right item — give it back
    mob.Command('say Interesting, but not what I need right now.');
    mob.Command('give !' + String(eventDetails.item.ItemId) + ' ' + user.ShorthandId());
    return true;
}
