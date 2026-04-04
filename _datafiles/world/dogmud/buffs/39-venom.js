function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'You feel venom seeping into your bloodstream!');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' winces as venom takes hold.', actor.UserId());
}

function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var dmg = Math.floor(maxHP * (0.08 + Math.random() * 0.04));
    if (dmg < 3) dmg = 3;
    actor.AddHealth(-dmg);
    SendUserMessage(actor.UserId(), 'The venom burns through your veins!');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' shudders as venom courses through them.', actor.UserId());
}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The venom finally subsides, leaving you weakened.');
}
