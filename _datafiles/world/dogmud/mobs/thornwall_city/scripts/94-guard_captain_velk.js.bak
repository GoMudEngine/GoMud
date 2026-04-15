// Guard Captain Velk - Mob 94
// Handles quest item delivery for Q10 and Q14.
// Q10: protection notice (item 30) → 10-report
// Q14: bribe ledger (item 40036) → 14-report

const PROTECTION_NOTICE_ID = 30;
const BRIBE_LEDGER_ID = 40036;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    // Protection notice delivery for Q10
    if ( eventDetails.item.ItemId == PROTECTION_NOTICE_ID ) {
        if ( user.HasQuest("10-start") && !user.HasQuest("10-report") ) {
            mob.Command('say A protection racket? Let me see that.', 1.0);
            mob.Command('say This is enough to act on. I will put a patrol on the back alleys and bring someone in for questioning.', 2.5);
            mob.Command('say Tell the tavern keeper he can stop paying. The watch will handle this now.', 4.5);
            user.GiveQuest("10-report");
            return true;
        }
        mob.Command('say I have already seen this. The patrols are out.');
        return true;
    }

    // Bribe ledger delivery for Q14
    if ( eventDetails.item.ItemId == BRIBE_LEDGER_ID ) {
        if ( user.HasQuest("14-evidence") && !user.HasQuest("14-report") ) {
            mob.Command('say A bribe ledger? Let me see that.', 1.0);
            mob.Command('say Names, dates, amounts. Gate captains, customs clerks, a warehouse inspector -- all on the take.', 2.5);
            mob.Command('say This changes everything. I am ordering arrests tonight.', 4.5);
            user.GiveQuest("14-report");
            return true;
        }
        mob.Command('say I already have the ledger. The arrests are underway.');
        return true;
    }

    // Not the right item — give it back
    mob.Command('say This is not evidence I can use. File a proper report.');
    mob.Command('give !' + String(eventDetails.item.ItemId) + ' ' + user.ShorthandId());
    return true;
}
