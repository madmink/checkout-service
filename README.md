# Checkout Service

Checkout Service is a Golang REST API for calculating checkout totals based on scanned product SKUs and active promotion rules.

The service supports:

* Product lookup by SKU
* Flexible promotion rules using JSONB
* Checkout order and checkout order item persistence
* Inventory deduction inside database transaction
* Request ID propagation for production debugging
* Optional New Relic integration
* Docker Compose local development
* Unit test and E2E test support

---

## 1. Service Overview

This service exposes a checkout API that receives scanned items, calculates the order total, applies active promotions, stores the checkout result, and decreases product inventory.

Main endpoint:

```http
POST /checkout-api/v1/checkout
```

Example request:

```json
{
  "items": [
    {
      "sku": "43N23P",
      "quantity": 1
    }
  ]
}
```

Example successful response:

```json
{
  "status": "OK",
  "data": {
    "is_success": true,
    "value": {
      "checkout_order_id": "fbfdbaf2-d7c1-4743-87e9-80bd33bcc3a7",
      "items": [
        {
          "checkout_order_item_id": "c7d22975-aac8-4edb-b792-8e1fc3e1bfd5",
          "product_id": "22222222-2222-2222-2222-222222222222",
          "sku": "43N23P",
          "name": "MacBook Pro",
          "quantity": 1,
          "unit_price_amount_minor": 539999,
          "subtotal_amount_minor": 539999,
          "final_subtotal_amount_minor": 539999,
          "currency": "USD"
        },
        {
          "checkout_order_item_id": "4a2820f7-d1b5-48c4-bb6b-fbaf3e2acc13",
          "product_id": "44444444-4444-4444-4444-444444444444",
          "sku": "234234",
          "name": "Raspberry Pi B",
          "quantity": 1,
          "unit_price_amount_minor": 3000,
          "subtotal_amount_minor": 3000,
          "discount_amount_minor": 3000,
          "applied_promotion_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
          "currency": "USD"
        }
      ],
      "applied_promotions": [
        {
          "promotion_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
          "promotion_name": "MacBook Pro Free Raspberry Pi B",
          "promotion_type": "free_item",
          "discount_amount_minor": 3000
        }
      ],
      "total_before_discount_amount_minor": 542999,
      "total_discount_amount_minor": 3000,
      "final_total_amount_minor": 539999,
      "currency": "USD"
    }
  },
  "accessTime": "05-06-2026 15:16:07"
}
```

---

## 2. Tech Stack

```text
Language      : Go
Database      : PostgreSQL
Container     : Docker, Docker Compose
Testing       : Go test, unit test, E2E test
Observability : New Relic, structured request logging
```

---

## 4. Layered Design

The service is split into clear layers.

### 4.1 Controller Layer

The controller initializes routing and maps HTTP endpoints to handlers.

Responsibilities:

* Register API routes
* Apply middleware
* Start HTTP server
* Keep route mapping separated from business logic

Example route intent:

```text
POST /checkout-api/v1/checkout -> checkoutHandler.Checkout
```

---

### 4.2 Handler Layer

The handler layer is responsible for HTTP request and response handling.

Responsibilities:

* Decode JSON request body
* Validate basic HTTP request format
* Read request headers such as `Log-Request-ID`
* Call service layer
* Format success and error responses

The handler should not contain business rules. It only adapts HTTP input into service input.

---

### 4.3 Service Layer

The service layer contains the main checkout use case.

Responsibilities:

* Validate checkout request
* Aggregate duplicate SKUs
* Load active promotions
* Detect free-item target SKU from promotion rules
* Load required products
* Validate product availability and currency
* Build checkout items
* Apply promotion rules
* Calculate totals
* Persist checkout order in transaction
* Deduct inventory

Main flow:

```text
1. Validate request items
2. Aggregate quantity by SKU
3. Load active promotions and rules
4. Add free-item target SKU into lookup list when needed
5. Load products by lookup SKUs
6. Validate products, inventory, and currency
7. Build initial checkout items from requested SKUs
8. Apply promotions
9. Recalculate totals from final checkout items
10. Insert checkout order
11. Insert checkout order items
12. Update product inventory
13. Commit transaction
14. Return checkout response
```

---

### 4.4 Repository Layer

The repository layer handles database access.

Responsibilities:

* Execute SQL queries
* Map query result into entity structs
* Insert checkout order and items
* Update inventory
* Support transaction executor through shared `DBExecutor`

The repository does not contain promotion business logic.

---

## 5. Database Design

The database consists of five main tables:

```text
1. products
2. promotions
3. promotion_rules
4. checkout_orders
5. checkout_order_items
```

---

## 6. Table Design Explanation

### 6.1 products

Stores product master data.

```sql
CREATE TABLE products (
    product_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    price_amount_minor BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    inventory_qty INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Column intent:

| Column               | Purpose                                        |
| -------------------- | ---------------------------------------------- |
| `product_id`         | Unique product identifier                      |
| `sku`                | Unique product code used by checkout request   |
| `name`               | Product display name                           |
| `price_amount_minor` | Product price in minor unit, for example cents |
| `currency`           | Currency code, for example `USD`               |
| `inventory_qty`      | Current available stock                        |
| `is_active`          | Determines whether product can be purchased    |
| `created_at`         | Product creation timestamp                     |
| `updated_at`         | Product update timestamp                       |

The service uses `price_amount_minor` to avoid floating-point calculation issues.

Example:

```text
4999 = USD 49.99
539999 = USD 5,399.99
```

---

### 6.2 promotions

Stores promotion metadata.

```sql
CREATE TABLE promotions (
    promotion_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promotion_name VARCHAR(150) NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Column intent:

| Column           | Purpose                                        |
| ---------------- | ---------------------------------------------- |
| `promotion_id`   | Unique promotion identifier                    |
| `promotion_name` | Human-readable promotion name                  |
| `type`           | Promotion type used by service logic           |
| `description`    | Promotion explanation                          |
| `is_active`      | Determines whether promotion should be applied |
| `created_at`     | Promotion creation timestamp                   |
| `updated_at`     | Promotion update timestamp                     |

Supported promotion types:

```text
free_item
buy_x_pay_y
bulk_percentage
```

---

### 6.3 promotion_rules

Stores promotion configuration as JSONB.

```sql
CREATE TABLE promotion_rules (
    promotion_rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promotion_id UUID NOT NULL,
    rule_config JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Column intent:

| Column              | Purpose                                              |
| ------------------- | ---------------------------------------------------- |
| `promotion_rule_id` | Unique rule identifier                               |
| `promotion_id`      | Reference to promotion                               |
| `rule_config`       | Flexible JSONB config parsed based on promotion type |
| `created_at`        | Rule creation timestamp                              |
| `updated_at`        | Rule update timestamp                                |

The use of `JSONB` makes the promotion rule flexible. Each promotion type can have different configuration without adding many nullable columns.

Example `free_item` rule:

```json
{
  "trigger_sku": "43N23P",
  "target_sku": "234234",
  "free_qty_per_trigger": 1
}
```

Example `buy_x_pay_y` rule:

```json
{
  "target_sku": "120P90",
  "buy_qty": 3,
  "pay_qty": 2
}
```

Example `bulk_percentage` rule:

```json
{
  "target_sku": "A304SD",
  "min_qty": 3,
  "discount_percentage": 10
}
```

---

### 6.4 checkout_orders

Stores checkout summary.

```sql
CREATE TABLE checkout_orders (
    checkout_order_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    total_before_discount_amount_minor BIGINT NOT NULL,
    total_discount_amount_minor BIGINT NOT NULL,
    final_total_amount_minor BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Column intent:

| Column                               | Purpose                                   |
| ------------------------------------ | ----------------------------------------- |
| `checkout_order_id`                  | Unique checkout order identifier          |
| `total_before_discount_amount_minor` | Sum of all item subtotals before discount |
| `total_discount_amount_minor`        | Total discount applied                    |
| `final_total_amount_minor`           | Final amount after discount               |
| `currency`                           | Checkout currency                         |
| `created_at`                         | Checkout creation timestamp               |

Formula:

```text
final_total_amount_minor = total_before_discount_amount_minor - total_discount_amount_minor
```

---

### 6.5 checkout_order_items

Stores checkout item details.

```sql
CREATE TABLE checkout_order_items (
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Column intent:

| Column                        | Purpose                                   |
| ----------------------------- | ----------------------------------------- |
| `checkout_order_item_id`      | Unique checkout item identifier           |
| `checkout_order_id`           | Reference to checkout order               |
| `product_id`                  | Reference to product                      |
| `sku`                         | Snapshot of product SKU at checkout time  |
| `name`                        | Snapshot of product name at checkout time |
| `quantity`                    | Purchased quantity                        |
| `unit_price_amount_minor`     | Unit price snapshot                       |
| `subtotal_amount_minor`       | Quantity multiplied by unit price         |
| `discount_amount_minor`       | Discount applied to this item             |
| `final_subtotal_amount_minor` | Item subtotal after discount              |
| `applied_promotion_id`        | Promotion applied to this item, nullable  |
| `currency`                    | Item currency                             |
| `created_at`                  | Checkout item creation timestamp          |

Formula:

```text
subtotal_amount_minor = unit_price_amount_minor * quantity
final_subtotal_amount_minor = subtotal_amount_minor - discount_amount_minor
```

---

## 7. Initial Data

### 7.1 Products

| SKU      | Product        | Price Minor | Display Price | Inventory |
| -------- | -------------- | ----------: | ------------: |----------:|
| `120P90` | Google Home    |      `4999` |     USD 49.99 |       999 |
| `43N23P` | MacBook Pro    |    `539999` |  USD 5,399.99 |       999 |
| `A304SD` | Alexa Speaker  |     `10950` |    USD 109.50 |       999 |
| `234234` | Raspberry Pi B |      `3000` |     USD 30.00 |       999 |

---

### 7.2 Promotions

| Promotion                              | Type              | Rule                                            |
| -------------------------------------- | ----------------- | ----------------------------------------------- |
| MacBook Pro Free Raspberry Pi B        | `free_item`       | Each MacBook gives 1 free Raspberry Pi B        |
| Google Home Buy 3 Pay 2                | `buy_x_pay_y`     | Buy 3 Google Homes, pay only 2                  |
| Alexa Speaker Bulk 10 Percent Discount | `bulk_percentage` | Buy at least 3 Alexa Speakers, get 10% discount |

---

## 8. Promotion Calculation Examples

### 8.1 Google Home Buy 3 Pay 2

Request:

```json
{
  "items": [
    {
      "sku": "120P90",
      "quantity": 3
    }
  ]
}
```

Calculation:

```text
Google Home unit price = 4999
Quantity               = 3

Subtotal               = 4999 * 3 = 14997
Discount               = 4999 * 1 = 4999
Final total            = 14997 - 4999 = 9998
```

Expected summary:

```json
{
  "total_before_discount_amount_minor": 14997,
  "total_discount_amount_minor": 4999,
  "final_total_amount_minor": 9998,
  "currency": "USD"
}
```

---

### 8.2 MacBook Pro Free Raspberry Pi B

Request:

```json
{
  "items": [
    {
      "sku": "43N23P",
      "quantity": 1
    }
  ]
}
```

The client only sends MacBook Pro. The service loads active promotion rules, detects that MacBook Pro gives a free Raspberry Pi B, then internally loads Raspberry Pi product data.

Calculation:

```text
MacBook Pro subtotal       = 539999 * 1 = 539999
Free Raspberry Pi subtotal = 3000 * 1 = 3000
Free Raspberry Pi discount = 3000

Total before discount      = 539999 + 3000 = 542999
Total discount             = 3000
Final total                = 542999 - 3000 = 539999
```

Expected summary:

```json
{
  "total_before_discount_amount_minor": 542999,
  "total_discount_amount_minor": 3000,
  "final_total_amount_minor": 539999,
  "currency": "USD"
}
```

Expected item behavior:

| SKU      | Product        | Quantity | Subtotal | Discount | Final Subtotal |
| -------- | -------------- | -------: | -------: | -------: | -------------: |
| `43N23P` | MacBook Pro    |        1 | `539999` |      `0` |       `539999` |
| `234234` | Raspberry Pi B |        1 |   `3000` |   `3000` |            `0` |

---

### 8.3 Alexa Speaker Bulk Discount

Request:

```json
{
  "items": [
    {
      "sku": "A304SD",
      "quantity": 3
    }
  ]
}
```

Calculation:

```text
Alexa Speaker unit price = 10950
Quantity                 = 3

Subtotal                 = 10950 * 3 = 32850
Discount                 = 32850 * 10 / 100 = 3285
Final total              = 32850 - 3285 = 29565
```

Expected summary:

```json
{
  "total_before_discount_amount_minor": 32850,
  "total_discount_amount_minor": 3285,
  "final_total_amount_minor": 29565,
  "currency": "USD"
}
```

---

### 8.4 No Promotion Example

Request:

```json
{
  "items": [
    {
      "sku": "234234",
      "quantity": 1
    }
  ]
}
```

Calculation:

```text
Raspberry Pi B unit price = 3000
Quantity                  = 1

Subtotal                  = 3000
Discount                  = 0
Final total               = 3000
```

Expected summary:

```json
{
  "total_before_discount_amount_minor": 3000,
  "total_discount_amount_minor": 0,
  "final_total_amount_minor": 3000,
  "currency": "USD"
}
```

---

## 9. Transaction Design

Checkout persistence uses a database transaction.

Transaction flow:

```text
BEGIN
  Insert checkout_orders
  Insert checkout_order_items
  Decrease product inventory by SKU
COMMIT
```

If any operation fails:

```text
ROLLBACK
```

This protects data consistency. For example, if the checkout order is inserted but inventory update fails, the transaction is rolled back so the system does not store a checkout that cannot be fulfilled.

---

## 10. Request ID Logging

The service supports request ID based logging for production debugging.

Header:

```http
Log-Request-ID: unique-request-id
```

Purpose:

* Trace one request across handler, service, repository, and logs (especially in case of multi microservice layered call)
* Make production issue investigation easier
* Help correlate API response, server log, and external monitoring

Example curl:

```bash
curl --location "http://localhost:4000/checkout-api/v1/checkout" \
  --header "Content-Type: application/json" \
  --header "Log-Request-ID: manual-test-001" \
  --data "{\"items\":[{\"sku\":\"120P90\",\"quantity\":3}]}"
```

Expected production behavior:

```text
request_id = Log-Request-ID header if provided
request_id = generated UUID if header is missing
```

The response should also return the same request ID in the response header so the client can report it when an issue happens.

---

## 11. New Relic Integration

New Relic integration is configurable and can be disabled locally.

Example config:

```json
{
  "newrelic": {
    "isActive": false,
    "name": "checkout-service",
    "license_key": "placeholder",
    "ingest_key": "",
    "region": "US"
  }
}
```

Intent:

| Config        | Purpose                             |
| ------------- | ----------------------------------- |
| `isActive`    | Enable or disable New Relic         |
| `name`        | Application name shown in New Relic |
| `license_key` | New Relic license key               |
| `ingest_key`  | Optional ingest key                 |
| `region`      | New Relic region                    |

For local development, keep:

```json
{"isActive" : false}
```

For production, set:

```json
{"isActive" : false}
```

and provide real New Relic credentials through environment-specific config or secret management.

---

## 12. Docker Compose Local Development

Docker Compose is used to simplify local development.

Expected services:

```text
checkout-service
checkout-db
```

Run locally:

```bash
docker compose up --build
```

Stop services:

```bash
docker compose down
```

Rebuild cleanly:

```bash
docker compose down -v
docker compose up --build
```

The database migration and seed data are prepared so the service can be tested immediately after container startup.

---

## 13. Local API Testing

### 13.1 Google Home Buy 3 Pay 2

```bash
curl --location "http://localhost:4000/checkout-api/v1/checkout" \
  --header "Content-Type: application/json" \
  --header "Log-Request-ID: test-google-home-001" \
  --data "{\"items\":[{\"sku\":\"120P90\",\"quantity\":3}]}"
```

Expected:

```text
total_before_discount_amount_minor = 14997
total_discount_amount_minor        = 4999
final_total_amount_minor           = 9998
```

---

### 13.2 MacBook Pro Free Raspberry Pi

```bash
curl --location "http://localhost:4000/checkout-api/v1/checkout" \
  --header "Content-Type: application/json" \
  --header "Log-Request-ID: test-macbook-001" \
  --data "{\"items\":[{\"sku\":\"43N23P\",\"quantity\":1}]}"
```

Expected:

```text
total_before_discount_amount_minor = 542999
total_discount_amount_minor        = 3000
final_total_amount_minor           = 539999
```

---

### 13.3 Alexa Speaker Bulk Discount

```bash
curl --location "http://localhost:4000/checkout-api/v1/checkout" \
  --header "Content-Type: application/json" \
  --header "Log-Request-ID: test-alexa-001" \
  --data "{\"items\":[{\"sku\":\"A304SD\",\"quantity\":3}]}"
```

Expected:

```text
total_before_discount_amount_minor = 32850
total_discount_amount_minor        = 3285
final_total_amount_minor           = 29565
```

---

## 14. Testing Strategy

### 14.1 Unit Test

Unit tests focus on service logic and use mocks instead of a real database.

Covered cases include:

```text
- Empty checkout request
- Invalid SKU
- Insufficient inventory
- Product repository failure
- Promotion repository failure
- Free item promotion
- Buy X Pay Y promotion
- Bulk percentage promotion
- Insert checkout order rollback
- Insert checkout order items rollback
- Inventory update rollback
- Commit failure
- No promotion scenario
```

Run unit tests:

```bash
go test ./app/service -v
```

Run all tests:

```bash
go test ./... -v
```

Run with coverage:

```bash
go test ./... -cover
```

---

### 14.2 E2E Test

E2E tests call the running API through HTTP.

E2E flow:

```text
HTTP request
  -> controller
  -> handler
  -> service
  -> repository
  -> PostgreSQL
  -> JSON response
```

Run the service first:

```bash
docker compose up --build
```

Then run E2E tests:

```bash
go test ./tests/e2e -v
```

---

## 15. CI

CI is build to ensure the project can be built and tested automatically.

Suggested CI checks:

```text
1. Checkout repository
2. Setup Go
3. Run go mod download
4. Run go test ./...
5. Build Docker image
```

Example GitHub Actions workflow:

```yaml
name: Checkout Service CI

on:
  push:
    branches:
      - main
  pull_request:

jobs:
  test-and-build:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout source code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Download dependencies
        run: go mod download

      - name: Run tests
        run: go test ./... -v

      - name: Build Docker image
        run: docker build -t checkout-service .
```

CI is a plus point because it proves the repository is runnable and testable outside the developer machine.

---

## 16. Configuration

application_docker.json is included intentionally for ease of use on github CI and local docker build, but the `license_key` is intentionally removed and the `is_active` configuration set to false for skipping the new relic integration

---

## 17. Design Notes

### 17.1 Flexible Promotion Rule Design

Promotion rules are stored in `JSONB` instead of fixed columns.

Reason:

```text
free_item needs trigger_sku, target_sku, free_qty_per_trigger
buy_x_pay_y needs target_sku, buy_qty, pay_qty
bulk_percentage needs target_sku, min_qty, discount_percentage
```

Using JSONB prevents many unused nullable columns and allows new promotion rule types to be added later with minimal table changes.

---

### 17.2 Currency-Aware Amount Design

Amounts are stored using integer minor units.

Example:

```text
USD 49.99 -> 4999
USD 30.00 -> 3000
```

Reason:

```text
Avoid floating-point precision issues in money calculation.
```

The service validates that checkout items use the same currency.

---

### 17.3 Free Item Lookup Design

For free item promotion, the free SKU may not be sent by the client.

Example request:

```json
{
  "items": [
    {
      "sku": "43N23P",
      "quantity": 1
    }
  ]
}
```

The service loads active promotion rules, detects the free target SKU `234234`, and adds it to product lookup internally.

This allows the response to include the free Raspberry Pi line item even when the client only scanned MacBook Pro.

---

### 17.4 Checkout Total Design

Order total is calculated from final checkout items after promotion is applied.

Formula:

```text
total_before_discount_amount_minor = sum(item.subtotal_amount_minor)
total_discount_amount_minor        = sum(item.discount_amount_minor)
final_total_amount_minor           = sum(item.final_subtotal_amount_minor)
```

This ensures:

```text
total_before_discount_amount_minor - total_discount_amount_minor = final_total_amount_minor
```

---


## 18. Summary


- Clean controller, handler, service, repository separation
- Flexible JSONB promotion rule design
- Transaction-safe checkout persistence
- Inventory update rollback on failure
- Request ID logging for issue tracing (supported across microservice layer)
- Optional New Relic observability
- Docker Compose local development
- Unit and E2E testing support
- CI-ready repository structure

