-- The Campfire message a prompt was posted as, parsed from the Location header
-- of the create response. Needed to disable the previous numbered prompt's
-- buttons, and to resolve a tap back to the prompt that printed it.
--
-- Nullable: a prompt whose send failed never got a message id, and phase 2's
-- rows predate the column entirely.
alter table prompts add column external_message_id text;

create unique index prompts_external_message_id_key
  on prompts (external_message_id) where external_message_id is not null;
