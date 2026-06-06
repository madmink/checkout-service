package query

const SelectProductBySKU = `
SELECT
    product_id,
    sku,
    name,
    price_amount_minor,
    currency,
    inventory_qty,
    is_active,
    created_at,
    updated_at
FROM products
WHERE sku = $1
  AND is_active = TRUE;
`

const SelectProductsBySKUs = `
SELECT
    product_id,
    sku,
    name,
    price_amount_minor,
    currency,
    inventory_qty,
    is_active,
    created_at,
    updated_at
FROM products
WHERE sku = ANY($1)
  AND is_active = TRUE;
`

const SelectActivePromotionsWithRules = `
SELECT
    p.promotion_id,
    p.promotion_name,
    p.type,
    p.description,
    p.is_active,
    p.created_at,
    p.updated_at,

    pr.promotion_rule_id,
    pr.promotion_id,
    pr.rule_config,
    pr.created_at,
    pr.updated_at
FROM promotions p
INNER JOIN promotion_rules pr
    ON pr.promotion_id = p.promotion_id
WHERE p.is_active = TRUE
ORDER BY p.created_at ASC;
`

const InsertCheckoutOrder = `
INSERT INTO checkout_orders (
    checkout_order_id,
    total_before_discount_amount_minor,
    total_discount_amount_minor,
    final_total_amount_minor,
    currency,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6
);
`

const InsertCheckoutOrderItems = `
INSERT INTO checkout_order_items (
    checkout_order_item_id,
    checkout_order_id,
    product_id,
    sku,
    name,
    quantity,
    unit_price_amount_minor,
    subtotal_amount_minor,
    discount_amount_minor,
    final_subtotal_amount_minor,
    applied_promotion_id,
    currency,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);
`

const UpdateInventoryBySKU = `
UPDATE products
SET
    inventory_qty = inventory_qty - $1,
    updated_at = $2
WHERE sku = $3
  AND inventory_qty >= $1
  AND is_active = TRUE;
`

const SelectCheckoutOrderByID = `
SELECT
    checkout_order_id,
    total_before_discount_amount_minor,
    total_discount_amount_minor,
    final_total_amount_minor,
    currency,
    created_at
FROM checkout_orders
WHERE checkout_order_id = $1;
`

const SelectCheckoutOrderItemsByCheckoutOrderID = `
SELECT
    checkout_order_item_id,
    checkout_order_id,
    product_id,
    sku,
    name,
    quantity,
    unit_price_amount_minor,
    subtotal_amount_minor,
    discount_amount_minor,
    final_subtotal_amount_minor,
    applied_promotion_id,
    currency,
    created_at
FROM checkout_order_items
WHERE checkout_order_id = $1
ORDER BY created_at ASC;
`
