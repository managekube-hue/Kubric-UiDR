# Entity Relationship Diagram

## User Account Registry (UAR) Schema

```
┌──────────────────────────┐
│        USERS             │
├──────────────────────────┤
│ id (UUID)          [PK]  │
│ username           [UNQ] │
│ email              [UNQ] │
│ blake3_fingerprint [UNQ] │
│ created_at              │
│ updated_at              │
│ is_active               │
│ metadata (JSONB)        │
└────────┬─────────────────┘
         │
         │ have many
         │
    ┌────▼─────────────────┐
    │   USER_ROLES         │
    ├──────────────────────┤
    │ user_id (FK)  [PK]   │
    │ role_id (FK)  [PK]   │
    │ assigned_at         │
    └────┬─────────────────┘
         │
         │ many-to-many
         │
    ┌────▼─────────────────┐
    │     ROLES            │
    ├──────────────────────┤
    │ id (UUID)      [PK]  │
    │ name           [UNQ] │
    │ description         │
    │ created_at          │
    │ updated_at          │
    └────┬─────────────────┘
         │
         │ have many
         │
    ┌────▼──────────────────┐
    │ ROLE_PERMISSIONS      │
    ├───────────────────────┤
    │ role_id (FK)   [PK]   │
    │ permission_id (FK)[PK]│
    └────┬──────────────────┘
         │
         │ many-to-many
         │
    ┌────▼──────────────────┐
    │    PERMISSIONS        │
    ├───────────────────────┤
    │ id (UUID)       [PK]  │
    │ name            [UNQ] │
    │ resource              │
    │ action                │
    │ description           │
    └───────────────────────┘

┌──────────────────────────┐
│      AUDIT_LOG           │
├──────────────────────────┤
│ id (UUID)          [PK]  │
│ user_id (FK)            │
│ action                  │
│ resource                │
│ resource_id             │
│ changes (JSONB)         │
│ timestamp               │
└──────────────────────────┘
```

## Key Relationships

1. **Users ↔ Roles**: Many-to-many relationship via USER_ROLES join table
2. **Roles ↔ Permissions**: Many-to-many relationship via ROLE_PERMISSIONS join table
3. **Audit Log**: Tracks all changes for compliance and debugging

## Indexes

- `idx_users_blake3`: Hardware fingerprint lookup
- `idx_users_email`: Email lookup
- `idx_user_roles_user`: List roles for user
- `idx_user_roles_role`: List users with role
- `idx_role_permissions_role`: List permissions for role
- `idx_audit_log_user`: Find user's actions
- `idx_audit_log_timestamp`: Timeline queries

## Row-Level Security (RLS)

- Users can only see their own record
- Roles and permissions visible to all authenticated users
- User-role mappings visible to user or admins
- Audit logs visible only to admins

---

Generated: 2026-02-12
