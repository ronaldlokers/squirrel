comment on table coach_answers is
    'Kept for up to 24 months from said_at, by decision — the horizon 0020 '
    'left unstated. Store.PurgeCoachAnswersBefore is the mechanism; nothing '
    'calls it on a schedule yet. Store.DeleteLatestCoachAnswer removes a '
    'single exchange on request — the chat command !unsay — for the one you '
    'regret before the horizon reaches it on its own.';
