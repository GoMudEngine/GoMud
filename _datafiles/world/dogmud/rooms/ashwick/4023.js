// Maren's Cottage - Room 4023
// Handles two discovery interactions (lore only, no quest):
// 1. Push/move/pull the loose stone → reveals cavity → gives letter
// 2. Recipe discovery flavor text on pickup from drawer

var LETTER_ITEM_ID = 40041;

function onCommand(cmd, rest, user, room) {

    if ( rest == null || rest == "" ) { return false; }
    var target = rest.toLowerCase();

    // Stone interaction: push/move/pull stone → letter
    if ( cmd == "push" || cmd == "move" || cmd == "pull"
         || cmd == "shift" || cmd == "press" ) {
        if ( target == "stone" || target == "loose stone"
             || target == "hearthstone" ) {

            // Already has the letter item
            if ( user.HasItemId(LETTER_ITEM_ID) ) {
                user.SendText('The loose stone shifts aside, revealing the empty cavity behind it. Whatever was hidden here has already been found.');
                return true;
            }

            // Discover the letter
            user.GiveItem(LETTER_ITEM_ID);
            user.SendText('You push the loose stone aside. It grinds against its neighbors and shifts inward, revealing a small cavity in the wall behind it. Inside, a folded letter rests against the smooth stone -- hidden deliberately, waiting for someone to look.');
            room.SendText(user.GetCharacterName(true)+' pushes aside a loose stone near the hearth, revealing a hidden cavity.', user.UserId());
            return true;
        }
    }

    return false;
}
