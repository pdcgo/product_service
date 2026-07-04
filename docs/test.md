# Product Service Testing

This documents (1) **what the test suite should cover** per RPC, and (2) **how we test**
(the harness + conventions). It is the running spec for `product_service`'s Go tests — keep it in
sync when you add or change a handler. Coverage is verified by
`cd product_service && go test ./... -p 1`.

Legend: ✅ covered · ⏳ deferred (tracked in §3) · ⚠ known gap / caveat.

## 1. Testing that should cover

### Product Related
1. **ProductCreate** (`product_create_test.go`, `product_create_gen_test.go`)
    - ✅ happy path — row persisted with all editable fields, `deleted=false`.
    - ✅ validation — `team_id=0` and empty `name` → `InvalidArgument`.
    - ✅ authenticated create generates the authoritative **product code** from the real inserted
      id (`TEAMCODE-ALIAS-P-X-HEXID`) and writes the `product_category` join.
2. **ProductCodeGenerate** (`product_code_generate_test.go`)
    - ✅ previews the next product's code (`H2O-GH-P-X-1`, hex id).
    - ✅ unauthenticated → `Unauthenticated`; `team_id=0` → `InvalidArgument`.
3. **ProductUpdate** (`product_update_test.go`)
    - ✅ happy path — editable fields (including zero values) written.
    - ✅ team-scope isolation — another team → `NotFound`.
    - ✅ validation — missing id / team_id / name → `InvalidArgument`.
    - ✅ category replace — re-assigning `category_id` leaves exactly one join row.
    - ✅ soft-deleted product → `NotFound`.
4. **ProductDelete** (`product_delete_test.go`)
    - ✅ soft-delete (`deleted=true`); re-delete → `NotFound`; another team → `NotFound`.
5. **ProductDetail** (`product_detail_test.go`)
    - ✅ all editable fields + team name + the assigned category.
    - ✅ team-scope isolation; validation; soft-deleted product → `NotFound`.
    - ✅ category **still resolves after the category is soft-deleted** (ProductDetail reads it via
      a raw join that bypasses the soft-delete scope — see CategoryDelete below).
6. **ProductList** (`product_list_test.go`)
    - ✅ team-scoped, newest-first, excludes deleted + other teams; name search; pagination;
      validation.
7. **ProductByIDs** (`product_by_ids_test.go`)
    - ✅ maps the requested ids → product data (`image ->> 0`, `team_id`).
    - ⚠ by design **no team-scope** and **no soft-delete filter** — it is an explicit id lookup
      (e.g. resolving products for historical orders). Pinned by a test, not a bug.
8. **ProductSearch** (`product_search_test.go`)
    - ✅ by name / by ref_id (the request `search` oneof); team scope; excludes deleted; limit caps.
    - ⚠ `limit` is applied unconditionally, so a request with `Limit=0` returns nothing — always
      pass a positive limit.
9. **ProductListExport** — ⏳ deferred (server-stream CSV + order/revenue aggregation).
10. **ProductDuplicate** — ⏳ deferred (handler is an unimplemented stub).

### Category Related
1. **CategoryCreate** (`category_create_test.go`) — ✅ root (level 0) + child (`parent.level+1`);
   parent becomes non-leaf; empty name → `InvalidArgument`; unknown parent → `NotFound`.
2. **CategoryUpdate** (`category_update_test.go`) — ✅ reparent with subtree level-shift;
   rename-only; cycle guard → `InvalidArgument`; unknown id → `NotFound`; `is_last` maintained on
   old + new parents.
3. **CategoryDelete** (`category_delete_test.go`) — ✅ **soft-delete** (`gorm.DeletedAt`): the
   category and its whole subtree (cascade) are hidden from listings (still present via
   `Unscoped`), the old parent's `is_last` is recomputed, and the `product_category` joins are
   **preserved** so already-inserted products keep resolving their category. Unknown id →
   `NotFound`.
4. **CategoryList** (`category_list_test.go`) — ✅ nested tree (roots + children), subtree by
   `parent_id`, flat list on `search`. Soft-deleted categories are auto-excluded (model query).

## 2. How we test

- **Harness**: `moretest.Suite` + **Postgres** via `moretest_mock.MockPostgresDatabase(&scenario)`
  and a `DbScenario` callback — never sqlite. The whole scenario runs inside **one transaction**, so
  order matters: put any op you expect to fail *at the DB level* last. Connect `NotFound` /
  `InvalidArgument` returned from a zero-row check do **not** abort the tx, so those subtests can run
  anywhere.
- **One `*_test.go` per handler**, in the external `product_test` / `category_test` packages.
- **Service under test**: `product.NewProductService(db)` / `category.NewCategoryService(db)`.
- **Migrations**: each test `db.AutoMigrate(...)` only the models it touches, plus stand-in structs
  for legacy tables this service does not own — `teamCodeRow` / `userTeamRow` (`teams` /
  `user_teams`, in `product_code_generate_test.go`) and `productCategoryRow` (the `product_category`
  join, in `helper_test.go` / `category_delete_test.go`).
- **Authenticated caller**: inject identity with `authCtx(t, userID)` (`helper_test.go`), which wraps
  `access_interceptors.SetIdentityToCtx`. Handlers read it via `GetIdentityFromCtx`.
- **Assertions**: field equality via a follow-up `db.First(...)` (use `.Unscoped()` to see
  soft-deleted rows); error codes via `connect.CodeOf(err)`.
- **Run**: `cd product_service && go test ./... -p 1` (serial). Single package/test:
  `go test ./product -run TestProductDetail`.

## 3. Known gaps / deferred (not silently dropped)

- **ProductListExport** — no test yet (streaming CSV + aggregation across
  order_items / products / teams).
- **ProductDuplicate** — unimplemented stub; implement and test together.
- **ProductByIDs / ProductSearch** — intentionally skip team-scoping (id / keyword lookups);
  ProductByIDs also skips the soft-delete filter. The behavior is pinned by tests, not "fixed."
- **ProductSearch `Limit`** — applied unconditionally (`Limit=0` ⇒ no rows). Candidate cleanup.
