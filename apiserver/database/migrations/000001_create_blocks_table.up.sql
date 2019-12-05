CREATE TABLE `blocks`
(
    `id`                      BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `block_hash`              CHAR(64)        NOT NULL,
    `accepting_block_id`      BIGINT UNSIGNED NULL,
    `version`                 INT             NOT NULL,
    `hash_merkle_root`        CHAR(64)        NOT NULL,
    `accepted_id_merkle_root` CHAR(64)        NOT NULL,
    `utxo_commitment`         CHAR(64)        NOT NULL,
    `timestamp`               DATETIME        NOT NULL,
    `bits`                    INT UNSIGNED    NOT NULL,
    `nonce`                   BIGINT UNSIGNED NOT NULL,
    `blue_score`              BIGINT UNSIGNED NOT NULL,
    `is_chain_block`          TINYINT         NOT NULL,
    `mass`                    BIGINT          NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_blocks_block_hash` (`block_hash`),
    INDEX `idx_blocks_timestamp` (`timestamp`),
    INDEX `idx_blocks_is_chain_block` (`is_chain_block`),
    CONSTRAINT `fk_blocks_accepting_block_id`
        FOREIGN KEY (`accepting_block_id`)
            REFERENCES `blocks` (`id`)
);