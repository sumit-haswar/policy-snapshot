# Synthetic fixtures

The registry starts with a valid revision 2 bundle. When fixture administration is enabled,
select a profile with:

```bash
curl -s -X PUT http://localhost:8082/__fixtures/active \
  -H 'content-type: application/json' \
  -d '{"profile":"unavailable"}'
```

| Profile | Behavior |
| --- | --- |
| `valid-v2` | Complete revision 2; standard-region exports become denied |
| `unavailable` | Returns 503 for the manifest |

Fixtures contain no production or proprietary policy data.
