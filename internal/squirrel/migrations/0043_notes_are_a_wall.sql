update items
   set state = 'open', state_at = null, held_because = null
 where kind = 'note'
   and state in ('kept', 'waiting', 'blocked', 'someday');
