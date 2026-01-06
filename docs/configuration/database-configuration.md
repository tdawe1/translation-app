# Database Configuration

GengoWatcher SaaS uses **PostgreSQL 17** for persistent data storage and **GORM 2.0** as the Object-Relational Mapper (ORM) in the Go backend.

## 1. Connection Pool Settings

To ensure optimal performance under load, you can configure the connection pool in the backend:

| Setting | Default | Description |
|---------|---------|-------------|
| `DB_MAX_OPEN_CONNS` | `25` | Max number of open connections to the DB. |
| `DB_MAX_IDLE_CONNS` | `10` | Max number of idle connections in the pool. |
| `DB_CONN_MAX_LIFETIME` | `1h` | Max amount of time a connection may be reused. |

---

## 2. Schema Management

We use **Alembic** (Python-based) for database migrations. This ensures schema consistency across development and production environments.

### Applying Migrations
```bash
cd backend
alembic upgrade head
```

### Creating New Migrations
If you modify the models in `backend/internal/models/`, you must create a new migration:
```bash
alembic revision --autogenerate -m "added_new_field_to_user"
```

---

## 3. Data Integrity & Types

- **UUIDs**: We use `v4 UUIDs` for all primary keys to ensure global uniqueness and prevent ID enumeration attacks.
- **Timestamps**: All tables include `created_at` and `updated_at` (UTC).
- **Soft Deletes**: Some tables use GORM's `DeletedAt` for soft-deletion of critical data.

---

## 4. Backup & Maintenance

### Daily Backups
In production, we recommend automated daily backups.
```bash
# Example pg_dump command
pg_dump -U gengo -h localhost gengowatcher > backup_$(date +%Y%m%d).sql
```

### Indexing
Ensure critical columns like `user_id`, `email`, and `token` are indexed. Our default migrations include these indexes. You can verify them with:
```sql
\d+ users;
```

---

## 5. Security (SSL/TLS)

In production, always set `DB_SSLMODE=require` to encrypt traffic between the API and the database.

**Production Connection String:**
`postgres://user:pass@host:5432/dbname?sslmode=require`

## Next Steps
- [Environment Variables](../configuration/environment-variables.md)
- [Database Schema Reference](../database/schema-reference.md)
