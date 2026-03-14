// Maren's Cottage - Room 4023
// Handles two discovery interactions:
// 1. Push/move/pull the loose stone → reveals cavity → gives letter
// 2. Get recipe from drawer → grants quest token (container handles item)

var LETTER_ITEM_ID = 40041;
var RECIPE_ITEM_ID = 40042;

function onCommand(cmd, rest, user, room) {

    if ( rest == null || rest == "" ) { return false; }
    var target = rest.toLowerCase();

    // Stone interaction: push/move/pull stone → letter
    if ( cmd == "push" || cmd == "move" || cmd == "pull"
         || cmd == "shift" || cmd == "press" ) {
        if ( target == "stone" || target == "loose stone"
             || target == "hearthstone" ) {

            // Already found the letter
            if ( user.HasQuest("17-letter") ) {
                user.SendText('The loose stone shifts aside, revealing the empty cavity behind it. Whatever was hidden here has already been found.');
                return true;
            }

            // Already has the letter item somehow
            if ( user.HasItemId(LETTER_ITEM_ID) ) {
                user.SendText('The cavity behind the stone is empty. You already have what was hidden here.');
                return true;
            }

            // Discover the letter
            user.GiveItem(LETTER_ITEM_ID);
            user.GiveQuest("17-letter");
            user.SendText('You push the loose stone aside. It grinds against its neighbors and shifts inward, revealing a small cavity in the wall behind it. Inside, a folded letter rests against the smooth stone -- hidden deliberately, waiting for someone to look.');
            room.SendText(user.GetCharacterName(true)+' pushes aside a loose stone near the hearth, revealing a hidden cavity.', user.UserId());
            return true;
        }
    }

    // Recipe discovery: quest token on pickup from drawer container
    if ( cmd == "get" || cmd == "take" ) {
        if ( target == "recipe" || target == "recipe page"
             || target == "page" || target == "herbalism recipe page" ) {
            if ( user.HasItemId(RECIPE_ITEM_ID)
                 && !user.HasQuest("17-recipe") ) {
                user.GiveQuest("17-recipe");
                user.SendText('The bedside table yields a single page of careful handwriting from its swollen drawer. You could use it to study the technique.');
            }
            return false; // Let engine handle container pickup
        }
    }

    return false;
}
