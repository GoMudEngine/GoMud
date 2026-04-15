// Records Clerk Pell - Mob 99
// Handles give-based quest item delivery for Quest 6 (The Collector's Burden)
// and Quest 9 (The Temple's Tithe Audit).

const REPORT_ITEM_ID = 31;
const LEDGER_ITEM_ID = 29;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    var itemId = eventDetails.item.ItemId;

    if ( itemId == REPORT_ITEM_ID ) {
        // Must be on quest 6
        if ( !user.HasQuest("6-start") ) {
            mob.Command('say I do not accept unsolicited documents. File a request first.');
            return true;
        }

        // Already filed
        if ( user.HasQuest("6-report") ) {
            mob.Command('say The report has already been filed. Reference number 4-7-12-B.');
            return true;
        }

        user.GiveQuest("6-report");
        mob.Command('say A maintenance report from the bridge keeper at Watchers Crossing. Let me see that.', 1.0);
        mob.Command('say Damage assessments, repair projections... this is thorough. I will file it with the magistrate. Reference number 4-7-12-B. Return to Harn and tell him it has been filed.', 3.0);
        return true;
    }

    // Tithe ledger for Quest 9 — Pell redirects to Olen
    if ( itemId == LEDGER_ITEM_ID ) {
        if ( user.HasQuest("9-start") && !user.HasQuest("9-end") ) {
            mob.Command('say The tithe ledger? I filed the copy. This is the original.', 1.0);
            mob.Command('say I am a clerk, not an auditor. Take it to Priest Olen at the temple -- he is the one who requested the investigation. He needs to see these numbers, not me.', 3.0);
            // Give the ledger back so the player can bring it to Olen
            user.GiveItem(LEDGER_ITEM_ID);
            return true;
        }
        mob.Command('say A tithe ledger? I have no open request for this. File a form first.');
        user.GiveItem(LEDGER_ITEM_ID);
        return true;
    }

    mob.Command('say If you need something filed, state its purpose. I do not accept random documents.');
    return true;
}
