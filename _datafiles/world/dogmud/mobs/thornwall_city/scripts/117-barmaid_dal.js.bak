
// Barmaid Dal — cycles between main tavern (472) and back corner (484)
// for flavor. Occasionally gets heckled by the old regulars.

var HOME_ROOM = 472;
var BACK_ROOM = 484;
var MIN_HOME_ROUNDS = 75;
var MIN_BACK_ROUNDS = 13;
var MAX_BACK_ROUNDS = 20;

var serviceEmotes = [
    'emote sets three cups on the scarred table without a word and collects the empties.',
    'emote tops off the cups on the table, ignoring the muttered commentary.',
    'emote drops a plate of stale bread on the table. "Eat something for once."'
];

var harassmentResponses = [
    ['emote nudges his cup forward with one finger, grinning.',
     'emote smacks the cup back without looking. "Ask nicer."'],
    ['say Dal, you get prettier every year.',
     'emote does not look up from the tray. "And you get older. Funny how that works."'],
    ['emote reaches for the bread and accidentally brushes her arm.',
     'emote pulls her arm back and gives him a look that could curdle milk.'],
    ['say How about a smile with that ale, Dal?',
     'emote sets the cup down hard enough to slosh. "How about a tip?"']
];

function onIdle(mob, room) {

    var roundNow = UtilGetRoundNumber();
    var currentRoom = mob.GetRoomId();

    // ── AT HOME (main tavern) ──
    if (currentRoom == HOME_ROOM) {

        var homeStart = mob.GetTempData("homeStart");
        if (homeStart == null) {
            mob.SetTempData("homeStart", roundNow);
            return false; // let normal idle commands run
        }

        var elapsed = roundNow - homeStart;
        if (elapsed < MIN_HOME_ROUNDS) {
            return false;
        }

        // After minimum wait, 15% chance per round to head to back room
        if (UtilDiceRoll(1, 100) <= 15) {
            mob.SetTempData("homeStart", null);
            mob.SetTempData("arrivedBack", roundNow);
            mob.SetTempData("backDuration",
                MIN_BACK_ROUNDS + UtilDiceRoll(1, MAX_BACK_ROUNDS - MIN_BACK_ROUNDS + 1) - 1);
            mob.Command("pathto " + String(BACK_ROOM));
            return true;
        }

        return false;
    }

    // ── AT BACK CORNER ──
    if (currentRoom == BACK_ROOM) {

        var arrivedBack = mob.GetTempData("arrivedBack");
        if (arrivedBack == null) {
            // Just arrived — service emote
            mob.SetTempData("arrivedBack", roundNow);
            mob.SetTempData("backDuration",
                MIN_BACK_ROUNDS + UtilDiceRoll(1, MAX_BACK_ROUNDS - MIN_BACK_ROUNDS + 1) - 1);

            var svc = serviceEmotes[UtilDiceRoll(1, serviceEmotes.length) - 1];
            mob.Command(svc);

            // 40% chance a patron heckles and she responds
            if (UtilDiceRoll(1, 100) <= 40) {
                var pair = harassmentResponses[UtilDiceRoll(1, harassmentResponses.length) - 1];
                // Pick a random patron to deliver the line
                var patrons = [114, 115, 116];
                var patronId = patrons[UtilDiceRoll(1, patrons.length) - 1];
                var patron = room.GetMob(patronId, true);
                if (patron != null) {
                    patron.Command(pair[0], 2.0);
                    mob.Command(pair[1], 4.0);
                }
            }

            return true;
        }

        var backElapsed = roundNow - arrivedBack;
        var duration = mob.GetTempData("backDuration");
        if (duration == null) duration = MIN_BACK_ROUNDS;

        if (backElapsed >= duration) {
            mob.SetTempData("arrivedBack", null);
            mob.SetTempData("backDuration", null);
            mob.SetTempData("homeStart", roundNow);
            mob.Command("pathto home");
            return true;
        }

        return false;
    }

    // ── EN ROUTE ──
    return false;
}

function onAsk(mob, room, eventDetails) {

    var gossipWords = ["gossip", "news", "rumor", "rumors", "talk", "word"];
    var match = UtilFindMatchIn(eventDetails.askText, gossipWords);

    if (match.found) {
        mob.Command('say You want gossip? Ask the old men in the back corner.');
        mob.Command('say They have more stories than sense.', 2.0);
        return true;
    }

    return false;
}
