// Windwarden Sylara — mob script
// Handles giving Spirit Fetishes after quest 12 completion
// Also gives 4 extra fetishes on first ask after quest completion

var FETISH_ITEM_ID = 40031;
var BONUS_KEY = 'sylara-bonus-fetishes-given';

function onAsk(mob, room, eventDetails) {

    var askWords = ["fetish", "component", "spirit", "summon", "more"];
    var match = UtilFindMatchIn(eventDetails.askText, askWords);

    if (!match.found) {
        return false;
    }

    var user = GetUser(eventDetails.sourceId);
    if (user == null) {
        return false;
    }

    // Must have completed quest 12
    if (!user.HasQuest("12-end")) {
        mob.Command('say The spirits do not know you yet. Prove yourself first.');
        return true;
    }

    // First time after quest: give 4 bonus fetishes (quest reward gave 1)
    var bonusGiven = user.GetMiscCharacterData(BONUS_KEY);
    if (!bonusGiven || bonusGiven === '' || bonusGiven === '0') {
        user.GiveItem(FETISH_ITEM_ID);
        user.GiveItem(FETISH_ITEM_ID);
        user.GiveItem(FETISH_ITEM_ID);
        user.GiveItem(FETISH_ITEM_ID);
        user.SetMiscCharacterData(BONUS_KEY, '1');
        mob.Command('emote reaches into a pouch and produces several small bundles of grass and wolf fur.');
        mob.Command('say Take these. The spirits provided well this season. You will need them for the calling.', 2.0);
        return true;
    }

    // Subsequent asks: give 1 fetish
    user.GiveItem(FETISH_ITEM_ID);
    mob.Command('emote pulls a spirit fetish from her pouch and holds it out.');
    mob.Command('say For the calling. The steppe provides.', 2.0);
    return true;
}
