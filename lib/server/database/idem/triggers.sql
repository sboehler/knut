DROP VIEW IF EXISTS accounts;
CREATE VIEW IF NOT EXISTS accounts AS 
SELECT id, name, open_date, close_date FROM accounts_history
WHERE created_at <= CURRENT_TIMESTAMP AND deleted_at > CURRENT_TIMESTAMP;

CREATE TRIGGER IF NOT EXISTS accounts_insert INSTEAD OF INSERT ON accounts
FOR EACH ROW
BEGIN
  INSERT INTO accounts_history(id, name, open_date, close_date)
    VALUES (new.id, new.name, new.open_date, new.close_date);
END;

CREATE TRIGGER IF NOT EXISTS accounts_update INSTEAD OF UPDATE ON accounts
FOR EACH ROW
BEGIN
  UPDATE accounts_history SET deleted_at = CURRENT_TIMESTAMP 
    WHERE id = new.id AND deleted_at = DATETIME('2999-12-31');
  INSERT INTO accounts_history(id, name, open_date, close_date)
    VALUES (new.id, new.name, new.open_date, new.close_date);
END;

CREATE TRIGGER IF NOT EXISTS accounts_delete INSTEAD OF DELETE ON accounts
FOR EACH ROW
BEGIN
  UPDATE accounts_history SET deleted_at = CURRENT_TIMESTAMP 
    WHERE id = old.id AND deleted_at = DATETIME('2999-12-31');
END;