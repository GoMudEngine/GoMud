function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'A cloud of toxic spores engulfs you!');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' is enveloped in a cloud of choking spores.', actor.UserId());
}

function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var dmg = Math.floor(maxHP * (0.05 + Math.random() * 0.03));
    if (dmg < 2) dmg = 2;
    actor.AddHealth(-dmg);
    SendUserMessage(actor.UserId(), 'The spores burn in your lungs, blurring your vision!');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' coughs and wheezes from the spore toxin.', actor.UserId());
}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'You finally cough up the last of the spores.');
}
