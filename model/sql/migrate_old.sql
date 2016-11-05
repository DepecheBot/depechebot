
drop table if exists chat2;
CREATE TABLE chat2 (
  primary_id INTEGER NOT NULL PRIMARY KEY,
  chat_id BIGINT UNIQUE NOT NULL,
  type TEXT NOT NULL,
  abandoned INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  user_name TEXT NOT NULL DEFAULT '',
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  open_time timestamptz NOT NULL,
  last_time DATETIME NOT NULL,
  state TEXT NOT NULL,
  params TEXT NOT NULL
);


insert into chat2
(primary_id, chat_id, type, abandoned, user_id, user_name, first_name, last_name, open_time, last_time, state, params)
select
primary_id, chat_id, type, abandoned, user_id, user_name, first_name, last_name, open_time, last_time, replace(state, 'parameter', 'param'), replace(groups, 'parameter', 'param')
from chat;

alter table chat rename to chat_old;
alter table chat2 rename to chat;
--drop table chat_old
