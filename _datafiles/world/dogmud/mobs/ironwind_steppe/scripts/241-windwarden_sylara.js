// Windwarden Sylara — mob script
// Handles:
//   onGive: wolf totem delivery for Q11
//   onAsk:  spirit fetish dispensing after Q12 completion

var TOTEM_ITEM_ID = 40033;
var FETISH_ITEM_ID = 40031;
var BONUS_KEY = 'sylara-bonus-fetishes-given';

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    // Wolf totem delivery for Q11
    if ( eventDetails.item.ItemId == TOTEM_ITEM_ID ) {
        if ( user.HasQuest("11-sylara") && !user.HasQuest("11-end") ) {
            mob.Command('say You found it.', 1.0);
            mob.Command('say The wolf spirit\'s voice -- I can hear it again.', 2.5);
            mob.Command('say The spirits noticed what you did. Ask me about the covenant when you are ready.', 4.0);
            user.GiveQuest("11-end");
            return true;
        }
        mob.Command('say The totem is where it belongs. Thank you.');
        return true;
    }

    // Not the right item
    mob.Command('say The steppe provides what I need. I have no use for this.');
    mob.Command('give !' + String(eventDetails.item.ItemId) + ' ' + user.ShorthandId());
    return true;
}

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
