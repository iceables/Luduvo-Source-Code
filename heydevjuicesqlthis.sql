CREATE TABLE assets (
    id INT NOT NULL AUTO_INCREMENT,
    uid INT NOT NULL,
    name VARCHAR(60) NOT NULL,
    description VARCHAR(500) DEFAULT NULL,
    type VARCHAR(10) NOT NULL,
    file_path VARCHAR(255) NOT NULL,
    approval_state VARCHAR(10) NOT NULL DEFAULT 'pending',
    reviewer_id INT DEFAULT NULL,
    review_note VARCHAR(500) DEFAULT NULL,
    created_at DATETIME NOT NULL,
    reviewed_at DATETIME DEFAULT NULL,
    PRIMARY KEY (id),
    KEY idx_assets_uid (uid),
    KEY idx_assets_state_type (approval_state, type),
    KEY idx_assets_reviewer (reviewer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;