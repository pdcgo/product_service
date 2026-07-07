# Product Service
This Is part of Submodule of [Warehouse Infra](https://github.com/pdcgo/warehouse_infra). In Warehouse Infra this is live in folder `./product_service`.<br>
This Service planned and Intended for replacing legacy Product System that exists in Warehouse Infra. Its planned for microservice and planned to more dependentless, separating domain purpose for better developing big and complex system that exists in Warehouse Infra.<br>
For now its just be candidate for Take over Product System In Warehouse Infra legacy.
Status for this development is still in progress and not completely take over legacy system.

1. for database schema related, read this [Database Schema](database-schema.md).
2. for testing that should cover and documentation about testing read this [Testing](test.md).
3. this service also follow protobuf guideline of this. [Warehouse Infra Proto Guideline](../../docs/proto-guideline.md)

## Authentication & Authorization.
1. Use v2 roling system. not legacy system. for complete reference read [this](../../user_service/docs/readme.md#authentication--authorization)
2. use interceptor that live in [here](../../user_service/access_interceptors/interceptor.go)
3. DON'T use legacy interface on [this](../../shared/interfaces/authorization_iface/authorization.go)

Implemented via the **v2 access interceptor** (`user_service/access_interceptors.NewAccessInterceptor`), attached to
both `ProductService` and `CategoryService` in `register.go`. Each request message declares a
`(role_base.v1.request_policy)`: the Product CRUD + `ProductCodeGenerate` are `allow_only_authenticated` with the
`team_id` field tagged `(role_base.v1.use_scope)` (caller must hold a role in that team; ROLE_ROOT/ROLE_ADMIN bypass);
the Category RPCs are `allow_only_authenticated` (global master data); the pre-existing read/search/map RPCs are
`allow_all` (preserve prior behavior). Handlers read the caller via `access_interceptors.GetIdentityFromCtx(ctx)`.
Tightening the writes to **specific roles** (e.g. selling OWNER/ADMIN/CS, ADMIN for category management) is the
deferred "Review Role authorization" item below.

## Connect RPC Spec.
`ProductService` heavyly depend `connect-rpc` to serve and creating apis and grpc. Why we use `connectrpc` because its can be two mode as pure grpc and grpc-web that interact like web. And also supported http2. This service have several rpc:

1. Product Management Related RPC (implemented — CRUD over the legacy `products` table)
    - Create Product that named `ProductCreate`
    - Update Product that named `ProductUpdate`
    - Delete Product that named `ProductDelete` (soft delete via the `deleted` column)
    - List Product that named `ProductList` (paginated, team-scoped; simple `repeated ProductListItem` + `PageInfo`)
    - Detail Product that named `ProductDetail` (single record with all editable fields; backs the edit form)
    - Product Code Generate `ProductCodeGenerate` (adopt how ref id product generate in legacy)

    Editable fields: `name`, `ref_id`, `images[]`, `description`, `markup_percent`, `cross_locked`. Each product is
    owned by `team_id`; every write is scoped to the owning team. Handlers live one-per-file in
    `product/product_{create,update,delete,list,detail}.go`, backed by the `product_models.Product` model and the
    `db_migrations/00001_create_products.sql` legacy-compat migration.

2. Product List for Fastest Ops RPC.<br>
    This rpc use for components that need load Product list fast. for example product picker component in frontend.
    1. this rpc named `ProductListSearch`.

3. Product Data By IDs RPC.<br>
    This rpc use for getting product data by ids.
    1. this rpc named `ProductByIds`.


3. Category Management Related RPC
    - Create Category that named `CategoryCreate`
    - Delete Category that named `CategoryDelete`
    - List Category that named `CategoryList`
    - Update Category that named `CategoryUpdate`
    

### Product Management RPC

## Product List for Fastest Ops RPC.
1. the product can be search by:
    - team id
    - product name
    - product code


## Product Data By IDs RPC
1. this rpc follow guideline [this](../../docs/proto-guideline.md#rule-rpc-that-load-data-by-ids).



### Category Management RPC
1. make `CategoryList` is for all authenticated. for further its used for shareable component picker too for accros the team.
2. rpc `CategoryDelete` is a **soft delete** for better compatibility with products already inserted.
    Implemented via `gorm.DeletedAt` on `product_models.Category` (column `deleted_at`, migration
    `db_migrations/00003_add_deleted_at_to_categories.sql`): the category and its whole subtree
    (cascade) are hidden from listings/pickers but stay in the table, and the `product_category`
    join rows are **preserved** so an already-inserted product still resolves its category
    (`ProductDetail` reads the name via a raw join that ignores the soft-delete scope). See
    [Testing](test.md).

## Not Todo yet
1. Review Role authorization and security

