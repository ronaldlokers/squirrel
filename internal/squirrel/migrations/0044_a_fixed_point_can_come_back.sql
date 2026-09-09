alter table moments add column if not exists every_weeks smallint;

alter table moments add constraint moments_every_weeks_sane
    check (every_weeks is null or (every_weeks >= 1 and every_weeks <= 52));
