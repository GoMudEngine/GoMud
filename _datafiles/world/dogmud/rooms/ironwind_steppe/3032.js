// Rocky Shelf - Room 3032
// Quest item pickup: Windstone Sample (item 40032)
// Player can take/get windstone if on quest 11 (after rhett step)

var SAMPLE_ITEM_ID = 40032;

function onCommand(cmd, rest, user, room) {

    if ( (cmd == "get" || cmd == "take" || cmd == "grab" || cmd == "chip" || cmd == "collect") && rest != null && rest != "" ) {
        var target = rest.toLowerCase();
        if ( target == "crystal" || target == "windstone" || target == "sample" || target == "windstone sample" ) {

            // Must be on quest 11 and have spoken to Rhett
            if ( !user.HasQuest("11-rhett") ) {
                user.SendText('The pale blue crystal is pretty, but you have no reason to chip it free.');
                return true;
            }

            // Already picked it up
            if ( user.HasItemId(SAMPLE_ITEM_ID) ) {
                user.SendText('You already have a windstone sample.');
                return true;
            }

            // Already completed this step
            if ( user.HasQuest("11-samples") ) {
                user.SendText('You already delivered windstone samples to Rhett.');
                return true;
            }

            user.GiveItem(SAMPLE_ITEM_ID);
            user.SendText('You chip a chunk of pale blue crystal free from the quartz vein. It hums in your hand, warm despite its glassy surface. Rhett will want to see this.');
            room.SendText(user.GetCharacterName(true)+' chips a piece of crystal from the rock face.', user.UserId());
            return true;
        }
    }

    return false;
}
