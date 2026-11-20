CREATE TABLE blocked_enrollments (
    id                    VARCHAR(255) NOT NULL,
    style                 VARCHAR(31)  NOT NULL,
    clear_on_authenticate BOOLEAN      NOT NULL DEFAULT FALSE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),

    CHECK (id != ''),
    CHECK (style != '')
);
