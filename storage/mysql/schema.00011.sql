ALTER TABLE enrollments DROP FOREIGN KEY enrollments_ibfk_2;
ALTER TABLE enrollments ADD CONSTRAINT enrollments_ibfk_2
    FOREIGN KEY (user_id, device_id)
    REFERENCES users (id, device_id)
    ON DELETE CASCADE ON UPDATE CASCADE;
