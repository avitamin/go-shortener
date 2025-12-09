-- удалить столбец is_deleted из таблицы urls
ALTER TABLE urls
DROP COLUMN is_deleted;