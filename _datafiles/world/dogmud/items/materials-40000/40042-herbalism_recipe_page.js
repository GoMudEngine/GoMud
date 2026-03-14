// Herbalism Recipe Page - Item 40042
// When used (read/studied), triggers foraging skill progression.
// The item is consumed on use (uses: 1) — the knowledge is absorbed.

function onUse(user, item, room) {

    user.SendText('You study the recipe page, tracing the careful handwriting and the teacher\'s corrections. The technique for fever-bark extraction is elegant in its simplicity -- lower heat, longer time, patience as an ingredient. Something clicks in your understanding of how plants yield their properties.');
    user.TrainSkill('foraging', 1);

    return true;
}
