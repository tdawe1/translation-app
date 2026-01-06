# Pagination

To ensure high performance and low latency, endpoints that return collections of data (like jobs or audit logs) are paginated.

## Usage

Pagination is controlled via query parameters:

| Parameter | Default | Maximum | Description |
|-----------|---------|---------|-------------|
| `page` | `1` | N/A | The page number to retrieve. |
| `per_page` | `20` | `100` | The number of items per page. |

**Example Request:**
```http
GET /api/v1/watcher/jobs?page=2&per_page=50
```

---

## Response Structure

Paginated responses wrap the results in a `data` array and include a `pagination` metadata object.

```json
{
  "data": [
    { "id": "item_1", "...": "..." },
    { "id": "item_2", "...": "..." }
  ],
  "pagination": {
    "total": 150,
    "page": 2,
    "per_page": 50,
    "total_pages": 3,
    "has_next": true,
    "has_prev": true
  }
}
```

---

## Field Definitions

- **`total`**: The total number of items available across all pages.
- **`page`**: The current page number.
- **`per_page`**: The number of items requested/returned per page.
- **`total_pages`**: Total number of pages based on `total` and `per_page`.
- **`has_next`**: Boolean indicating if a subsequent page exists.
- **`has_prev`**: Boolean indicating if a previous page exists.

---

## Best Practices

### Sequential Access
When fetching all items, start at `page=1` and continue until `has_next` is `false`.

### Stable Sorting
Unless otherwise specified, results are sorted by `created_at` in descending order.

### Large Collections
For extremely large collections, it is more efficient to use the `per_page=100` setting to reduce the total number of round-trip requests.

## Next Steps
- [API Overview](../api/overview.md)
- [Rate Limiting](../api/rate-limiting.md)
