# zest overview

[zeke.notion.site/zest](https://zeker.notion.site/zest-5f76df8389e24e67b62b7c717daad6ab)

# backend development

[zeke.notion.site/zest-backend](https://zeker.notion.site/zest-backend-00f3a9b001bd44d38c7acc74f8738a4d)

## migrations

Working on getting https://atlasgo.io/docs setup.

```bash
curl -sSf https://atlasgo.sh | sh
```

To autogenerate a schema, can do:

```bash
atlas schema inspect -u "postgres://zeke:reyna@localhost:5432/zest?sslmode=disable" --format "{{ sql . }}"
```

To make a migration, can do:
```bash
make dev # atlas needs a dev database to plan changes
atlas schema apply -u "postgres://zeke:reyna@localhost:5432/zest?sslmode=disable" --to file://schema.sql --dev-url "postgres://atlas:pass@localhost:5444/postgres?sslmode=disable"
```