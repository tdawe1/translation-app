# Database Performance Optimization

As the number of users and jobs in GengoWatcher SaaS grows, database performance becomes critical. We use several strategies to keep queries fast and efficient.

## 1. Indexing Strategy

Indexes are the primary tool for speed. We prioritize indexes on columns used in `WHERE`, `JOIN`, and `ORDER BY` clauses.

| Table | Indexed Columns | Purpose |
|-------|-----------------|---------|
| `users` | `email` (Unique) | Fast login/lookup. |
| `found_jobs` | `user_id`, `status` | Dashboard filtering. |
| `found_jobs` | `found_at` | Sorting by recency. |
| `oauth_accounts`| `provider_user_id` | OAuth callback performance. |

---

## 2. Query Optimization

### A. Eager Loading
To avoid the "N+1 Problem", we use GORM's `.Preload()` to fetch related data in a single query.
```go
// Fetch user with their OAuth accounts in one query
db.Preload("OAuthAccounts").Find(&users)
```

### B. Partial Indexes
For tables with many rows, we use partial indexes to index only relevant data.
```sql
CREATE INDEX idx_active_watchers ON watcher_states(user_id) WHERE status = 'running';
```

---

## 3. Database Maintenance

### Vacuuming
PostgreSQL uses MVCC, which can lead to "bloat". Ensure `autovacuum` is enabled in production to reclaim space and update statistics.

### Analyzing
Regularly run `ANALYZE` to help the query planner choose the most efficient path.

---

## 4. Connection Management

We use **PgBouncer** as a transaction-mode connection pooler. This allows thousands of concurrent Go routines to share a smaller pool of physical database connections, significantly reducing overhead.

---

## 5. Scaling Out (Read Replicas)

For heavy dashboard usage, we implement **Read-Write Splitting**:
1. **Primary**: All `POST`, `PUT`, `DELETE` operations.
2. **Replicas**: All `GET` requests for job history and metrics.

## Next Steps
- [Backup & Recovery](../database/backup-recovery.md)
- [Relationships & ERD](../database/relationships.md)
