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


DROP VIEW IF EXISTS prices;
CREATE VIEW IF NOT EXISTS prices AS
SELECT date, commodity_id, target_commodity_id, price
FROM prices_history
WHERE created_at <= CURRENT_TIME AND deleted_at > CURRENT_TIMESTAMP;

CREATE TRIGGER IF NOT EXISTS prices_insert INSTEAD OF INSERT ON prices
FOR EACH ROW
WHEN NOT EXISTS (
    SELECT 1
    FROM prices
    WHERE date = new.date 
    AND commodity_id = new.commodity_id
    AND target_commodity_id = new.target_commodity_id 
    AND price = new.price)
BEGIN
  UPDATE prices_history SET deleted_at = CURRENT_TIMESTAMP
    WHERE date = new.date 
    AND commodity_id = new.commodity_id
    AND target_commodity_id = new.target_commodity_id;
  INSERT INTO prices_history(date, commodity_id, target_commodity_id, price)
    VALUES(new.date, new.commodity_id, new.target_commodity_id, new.price);
END;

CREATE TRIGGER IF NOT EXISTS prices_delete INSTEAD OF DELETE ON prices
FOR EACH ROW
BEGIN
  UPDATE prices_history 
    SET deleted_at = CURRENT_TIMESTAMP 
    WHERE date = old.date 
    AND commodity_id = old.commodity_id
    AND target_commodity_id = old.target_commodity_id
    AND deleted_at = DATETIME('2999-12-31');
END;