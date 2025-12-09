-- добавить столбец is_deleted в таблицу urls
ALTER TABLE urls
ADD COLUMN is_deleted BOOLEAN DEFAULT FALSE;