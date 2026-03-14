// Maren's Cottage - Room 4023
// Grants quest discovery tokens when players find hidden items.
// The letter (40041) is in the cavity container (behind the loose stone).
// The recipe (40042) is in the drawer container.
// The engine handles the actual item pickup from containers.
// This script adds quest tokens and flavor text.
// Skill progression is handled by the recipe item's onUse script.

var LETTER_ITEM_ID = 40041;
var RECIPE_ITEM_ID = 40042;

function onCommand(cmd, rest, user, room) {

    if ( cmd != "get" && cmd != "take" ) { return false; }
    if ( rest == null || rest == "" ) { return false; }

    var target = rest.toLowerCase();

    // Letter discovery (from cavity container)
    if ( target == "letter" || target == "creased letter"
         || target == "creased" ) {
        if ( user.HasItemId(LETTER_ITEM_ID)
             && !user.HasQuest("17-letter") ) {
            user.GiveQuest("17-letter");
            user.SendText('The cavity behind the loose stone holds a folded letter. The paper is creased from years of being read and refolded. Someone hid this here deliberately.');
        }
        return false;
    }

    // Recipe discovery (from drawer container)
    if ( target == "recipe" || target == "recipe page"
         || target == "page" || target == "herbalism recipe page" ) {
        if ( user.HasItemId(RECIPE_ITEM_ID)
             && !user.HasQuest("17-recipe") ) {
            user.GiveQuest("17-recipe");
            user.SendText('The drawer yields a single page of careful handwriting. Use it to study the technique.');
        }
        return false;
    }

    return false;
}
