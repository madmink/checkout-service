CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =========================================================
-- PRODUCTS
-- =========================================================

CREATE TABLE IF NOT EXISTS products (
                                        product_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    price_amount_minor BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    inventory_qty INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_products_price_amount_minor_non_negative
    CHECK (price_amount_minor >= 0),

    CONSTRAINT chk_products_inventory_qty_non_negative
    CHECK (inventory_qty >= 0)
    );

INSERT INTO products (
    product_id,
    sku,
    name,
    price_amount_minor,
    currency,
    inventory_qty,
    is_active
)
VALUES
    (
        '11111111-1111-1111-1111-111111111111',
        '120P90',
        'Google Home',
        4999,
        'USD',
        999,
        TRUE
    ),
    (
        '22222222-2222-2222-2222-222222222222',
        '43N23P',
        'MacBook Pro',
        539999,
        'USD',
        999,
        TRUE
    ),
    (
        '33333333-3333-3333-3333-333333333333',
        'A304SD',
        'Alexa Speaker',
        10950,
        'USD',
        999,
        TRUE
    ),
    (
        '44444444-4444-4444-4444-444444444444',
        '234234',
        'Raspberry Pi B',
        3000,
        'USD',
        999,
        TRUE
    )
    ON CONFLICT (sku) DO UPDATE SET
    name = EXCLUDED.name,
                             price_amount_minor = EXCLUDED.price_amount_minor,
                             currency = EXCLUDED.currency,
                             inventory_qty = EXCLUDED.inventory_qty,
                             is_active = EXCLUDED.is_active,
                             updated_at = CURRENT_TIMESTAMP;


-- =========================================================
-- PROMOTIONS
-- =========================================================

CREATE TABLE IF NOT EXISTS promotions (
                                          promotion_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promotion_name VARCHAR(150) NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS promotion_rules (
                                               promotion_rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promotion_id UUID NOT NULL,
    rule_config JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_promotion_rules_promotion
    FOREIGN KEY (promotion_id)
    REFERENCES promotions(promotion_id)
    ON DELETE CASCADE
    );

CREATE INDEX IF NOT EXISTS idx_promotions_type
    ON promotions(type);

CREATE INDEX IF NOT EXISTS idx_promotions_is_active
    ON promotions(is_active);

CREATE INDEX IF NOT EXISTS idx_promotion_rules_promotion_id
    ON promotion_rules(promotion_id);


-- =========================================================
-- PROMOTION SEED DATA
-- =========================================================

INSERT INTO promotions (
    promotion_id,
    promotion_name,
    type,
    description,
    is_active
)
VALUES
    (
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1',
        'MacBook Pro Free Raspberry Pi B',
        'free_item',
        'Each sale of a MacBook Pro comes with a free Raspberry Pi B.',
        TRUE
    ),
    (
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2',
        'Google Home Buy 3 Pay 2',
        'buy_x_pay_y',
        'Buy 3 Google Homes for the price of 2.',
        TRUE
    ),
    (
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3',
        'Alexa Speaker Bulk 10 Percent Discount',
        'bulk_percentage',
        'Buying at least 3 Alexa Speakers will get a 10 percent discount on all Alexa speakers.',
        TRUE
    )
    ON CONFLICT (promotion_id) DO UPDATE SET
    promotion_name = EXCLUDED.promotion_name,
                                      type = EXCLUDED.type,
                                      description = EXCLUDED.description,
                                      is_active = EXCLUDED.is_active,
                                      updated_at = CURRENT_TIMESTAMP;

INSERT INTO promotion_rules (
    promotion_rule_id,
    promotion_id,
    rule_config
)
VALUES
    (
        'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1',
        '{
            "trigger_sku": "43N23P",
            "target_sku": "234234",
            "free_qty_per_trigger": 1
        }'::jsonb
    ),
    (
        'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb2',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2',
        '{
            "target_sku": "120P90",
            "buy_qty": 3,
            "pay_qty": 2
        }'::jsonb
    ),
    (
        'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb3',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3',
        '{
            "target_sku": "A304SD",
            "min_qty": 3,
            "discount_percentage": 10
        }'::jsonb
    )
    ON CONFLICT (promotion_rule_id) DO UPDATE SET
    promotion_id = EXCLUDED.promotion_id,
                                           rule_config = EXCLUDED.rule_config,
                                           updated_at = CURRENT_TIMESTAMP;


-- =========================================================
-- CHECKOUT ORDERS
-- =========================================================

CREATE TABLE IF NOT EXISTS checkout_orders (
                                               checkout_order_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    total_before_discount_amount_minor BIGINT NOT NULL,
    total_discount_amount_minor BIGINT NOT NULL,
    final_total_amount_minor BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_checkout_orders_total_before_discount_non_negative
    CHECK (total_before_discount_amount_minor >= 0),

    CONSTRAINT chk_checkout_orders_total_discount_non_negative
    CHECK (total_discount_amount_minor >= 0),

    CONSTRAINT chk_checkout_orders_final_total_non_negative
    CHECK (final_total_amount_minor >= 0)
    );

CREATE TABLE IF NOT EXISTS checkout_order_items (
                                                    checkout_order_item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    checkout_order_id UUID NOT NULL,
    product_id UUID NOT NULL,
    sku VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    quantity INT NOT NULL,
    unit_price_amount_minor BIGINT NOT NULL,
    subtotal_amount_minor BIGINT NOT NULL,
    discount_amount_minor BIGINT NOT NULL,
    final_subtotal_amount_minor BIGINT NOT NULL,
    applied_promotion_id UUID NULL,
    currency VARCHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_checkout_order_items_order
    FOREIGN KEY (checkout_order_id)
    REFERENCES checkout_orders(checkout_order_id)
    ON DELETE CASCADE,

    CONSTRAINT fk_checkout_order_items_product
    FOREIGN KEY (product_id)
    REFERENCES products(product_id),

    CONSTRAINT fk_checkout_order_items_applied_promotion
    FOREIGN KEY (applied_promotion_id)
    REFERENCES promotions(promotion_id),

    CONSTRAINT chk_checkout_order_items_quantity_positive
    CHECK (quantity > 0),

    CONSTRAINT chk_checkout_order_items_unit_price_non_negative
    CHECK (unit_price_amount_minor >= 0),

    CONSTRAINT chk_checkout_order_items_subtotal_non_negative
    CHECK (subtotal_amount_minor >= 0),

    CONSTRAINT chk_checkout_order_items_discount_non_negative
    CHECK (discount_amount_minor >= 0),

    CONSTRAINT chk_checkout_order_items_final_subtotal_non_negative
    CHECK (final_subtotal_amount_minor >= 0)
    );

CREATE INDEX IF NOT EXISTS idx_checkout_order_items_checkout_order_id
    ON checkout_order_items(checkout_order_id);

CREATE INDEX IF NOT EXISTS idx_checkout_order_items_sku
    ON checkout_order_items(sku);

CREATE INDEX IF NOT EXISTS idx_checkout_order_items_applied_promotion_id
    ON checkout_order_items(applied_promotion_id);

