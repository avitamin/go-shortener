-- добавить столбец user_id в таблицу urls
alter table urls
    add column user_id varchar(255);