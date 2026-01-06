# Database Migrations

GengoWatcher SaaS uses **Alembic** to manage database schema evolutions. Migrations ensure that every environment (development, staging, production) stays in sync.

## 1. Directory Structure

Migrations are located in the `backend/alembic/` directory:
- `versions/`: Contains the individual migration scripts.
- `env.py`: The entry point for GORM/Alembic integration.
- `alembic.ini`: Configuration file.

---

## 2. Common Commands

### Applying Migrations
Update your database to the latest version:
```bash
alembic upgrade head
```

### Rolling Back
Revert the last migration:
```bash
alembic downgrade -1
```

### Creating a New Migration
When you change the `internal/models/` Go code, generate a new migration:
```bash
alembic revision --autogenerate -m "add_tier_to_users"
```

---

## 3. Best Practices

### A. Never Edit Migrations Manually
Generated files should only be edited if Alembic fails to detect a complex change (like a column rename).

### B. One Feature per Migration
Keep migrations focused. Don't mix unrelated schema changes in a single version file.

### C. Data Migrations
If you need to transform existing data, create a separate revision and use the `op.execute()` method to run raw SQL.

### D. Production Safety
Always test migrations on a copy of production data before applying them. For large tables (>1M rows), be aware that adding columns with default values or creating indexes can lock the table.

---

## 4. Troubleshooting

### "Target database is not at version..."
This happens if your `alembic_version` table is out of sync. Use `alembic stamp head` only if you are absolutely sure the schema matches your current code.

### "Can't locate revision..."
You may have merge conflicts in your migrations. Ensure every developer pulls the latest migrations before creating new ones.

## Next Steps
- [Schema Reference](../database/schema-reference.md)
- [Performance Optimization](../database/performance-optimization.md)
