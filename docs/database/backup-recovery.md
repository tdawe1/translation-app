# Backup & Recovery Procedures

Protecting user data is our highest priority. We employ a multi-layered backup strategy to ensure data can be recovered in the event of hardware failure, accidental deletion, or cyber-attack.

## 1. Backup Strategy

### A. Automated Snapshots
If using a managed service (e.g., AWS RDS), we enable **Daily Snapshots** with a 30-day retention period.

### B. Point-in-Time Recovery (PITR)
Enable **Write-Ahead Logging (WAL)** archiving to allow recovery to any specific second within the last 7 days.

---

## 2. Manual Backup (pg_dump)

For local development or custom migrations, you can manually export the database:

```bash
pg_dump -h $DB_HOST -U $DB_USER -d gengowatcher > backup_$(date +%Y%m%d).sql
```

To compress the backup:
```bash
pg_dump -U gengo gengowatcher | gzip > backup.sql.gz
```

---

## 3. Recovery Procedure

### A. Restoring from a Snapshot (AWS RDS Example)
1. Go to the RDS console.
2. Select the snapshot.
3. Click **Restore Snapshot**.
4. Configure the new instance (must have a different name).
5. Once running, update your `DATABASE_URL` to point to the new instance.

### B. Restoring from a SQL Dump
1. Create a fresh database:
   ```bash
   createdb -U gengo gengowatcher_restore
   ```
2. Import the dump:
   ```bash
   psql -U gengo -d gengowatcher_restore < backup.sql
   ```

---

## 4. Verification

Backups are useless if they don't work. We perform a **Recovery Drill** once every quarter:
1. Restore the latest backup to a staging environment.
2. Run automated tests to verify data integrity.
3. Document any issues found.

---

## 5. Security

- **Encryption**: All backups must be encrypted at rest using AES-256.
- **Off-site Storage**: Copy critical snapshots to a different geographical region (e.g., AWS US-West-2 to US-East-1).
- **Access Control**: Limit who can delete or modify backup files.

## Next Steps
- [Performance Optimization](../database/performance-optimization.md)
- [Schema Reference](../database/schema-reference.md)
