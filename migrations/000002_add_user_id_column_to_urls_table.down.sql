-- удалить столбец user_id из таблицы urls
alter table urls
    drop column user_id;