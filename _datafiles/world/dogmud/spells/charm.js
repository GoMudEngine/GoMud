// Charm spell — bend a creature's will to yours via an opposed charisma roll.
// School: manifestation. Cost: 120 conviction. Type: harmsingle.
// Harder against creatures already in combat. Stronger creatures resist more.

function onCast(sourceActor, targetActor) {
    // Companion cap check
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(),
            'You have reached your limit of bound companions. ' +
            'Release one before attempting another charm.');
        return false;
    }

    // Must have a valid mob target
    if (!targetActor || targetActor.MobTypeId() === 0) {
        SendUserMessage(sourceActor.UserId(),
            'You can only charm creatures, not people.');
        return false;
    }

    // Charm-immune check
    if (targetActor.IsCharmImmune()) {
        SendUserMessage(sourceActor.UserId(),
            'That creature\'s mind is beyond your reach — it is ' +
            'immune to charm.');
        return false;
    }

    SendUserMessage(sourceActor.UserId(),
        'You fix your gaze on your target, pressing your will ' +
        'against its mind...');
    SendRoomMessage(sourceActor.GetRoomId(),
        sourceActor.GetCharacterName(true) +
        ' stares intently at ' + targetActor.GetCharacterName(true) +
        ', brow furrowed with fierce concentration.',
        sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    var msgs = [
        'You lock eyes with your target, your will pressing ' +
            'against theirs...',
        'The air crackles with psychic tension...',
        'You feel the resistance wavering...'
    ];
    var pick = msgs[Math.floor(Math.random() * msgs.length)];
    SendUserMessage(sourceActor.UserId(), pick);
}

function onMagic(sourceActor, targetActor) {
    // Re-check cap (state may have changed while casting)
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(),
            'You have reached your limit of bound companions. ' +
            'The spell fades unfinished.');
        return;
    }

    if (!targetActor || targetActor.MobTypeId() === 0) {
        SendUserMessage(sourceActor.UserId(),
            'Your target is gone. The charm dissipates.');
        return;
    }

    if (targetActor.IsCharmImmune()) {
        SendUserMessage(sourceActor.UserId(),
            'That creature\'s mind is beyond your reach. ' +
            'The charm shatters.');
        return;
    }

    // Build attack score: charisma + manifestation skill bonus
    var attackScore = sourceActor.GetStat('charisma') +
        sourceActor.GetSkillLevel('manifestation') * 25;

    // Build defense score: willpower + 10% of total stat training pool
    var targetPool = targetActor.GetStatTrainingTotal();
    var defenseScore = targetActor.GetStat('willpower') +
        Math.round(targetPool * 0.10);

    // Aggro penalty: harder to charm something already fighting
    if (targetActor.IsAggroed()) {
        if (targetActor.GetAggroUserId() === sourceActor.UserId()) {
            // It's fighting us — steepest penalty
            attackScore = Math.round(attackScore * 0.75);
        } else {
            // It's fighting someone else — moderate penalty
            attackScore = Math.round(attackScore * 0.85);
        }
    }

    var success = sourceActor.RollOpposed(attackScore, defenseScore);

    if (success) {
        var targetName = targetActor.GetCharacterName(false);
        var targetNameTagged = targetActor.GetCharacterName(true);

        // Charm the mob permanently (99999 rounds — duration managed by
        // CharmDuration field set below)
        targetActor.CharmSet(sourceActor.UserId(), 99999);

        // Register as companion
        sourceActor.AddCompanion(
            targetActor.InstanceId(),
            'charmed',
            targetName
        );

        // Set timed charm duration based on caster stats
        var cha = sourceActor.GetStat('charisma');
        var skill = sourceActor.GetSkillLevel('manifestation');
        var duration = 50 + Math.floor(cha / 2) + skill * 3;
        sourceActor.SetCompanionCharmDuration(
            targetActor.InstanceId(),
            duration
        );

        SendUserMessage(sourceActor.UserId(),
            targetName + '\'s eyes glaze as your will takes hold. ' +
            'It is yours.');
        SendRoomMessage(sourceActor.GetRoomId(),
            sourceActor.GetCharacterName(true) +
            ' bends ' + targetNameTagged + ' to their will!',
            sourceActor.UserId());
    } else {
        var targetName = targetActor.GetCharacterName(false);
        var targetNameTagged = targetActor.GetCharacterName(true);

        SendUserMessage(sourceActor.UserId(),
            'You reach for ' + targetName + '\'s mind, but its will ' +
            'is iron. The spell shatters.');
        SendRoomMessage(sourceActor.GetRoomId(),
            sourceActor.GetCharacterName(true) +
            '\'s charm spell breaks against ' +
            targetNameTagged + '\'s resolve!',
            sourceActor.UserId());
    }
}
