// Kill Ground - Room 3082
// Quest item pickup: Carved Wolf Totem (item 40033)
// Player can take/get the totem if on quest 11 (after sylara step)

var TOTEM_ITEM_ID = 40033;

function onCommand(cmd, rest, user, room) {

    if ( (cmd == "get" || cmd == "take" || cmd == "grab") && rest != null && rest != "" ) {
        var target = rest.toLowerCase();
        if ( target == "totem" || target == "wolf totem" || target == "carved totem" || target == "carving" ) {

            // Must be on quest 11 and have spoken to Sylara
            if ( !user.HasQuest("11-sylara") ) {
                user.SendText('You see a small wooden carving among the bones, but it means nothing to you.');
                return true;
            }

            // Already picked it up
            if ( user.HasItemId(TOTEM_ITEM_ID) ) {
                user.SendText('You already have the wolf totem.');
                return true;
            }

            // Already completed this step
            if ( user.HasQuest("11-totem") ) {
                user.SendText('You already returned the totem to Sylara.');
                return true;
            }

            user.GiveItem(TOTEM_ITEM_ID);
            user.SendText('You pull the carved wolf totem free from the debris, brushing off dirt and old blood. The cedar wood is warm in your hand and smells faintly of sage. Sylara will want this back.');
            room.SendText(user.GetCharacterName(true)+' pulls a small wooden carving from among the bones.', user.UserId());
            return true;
        }
    }

    return false;
}
