# Database Migrations

This project uses [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations.

## Creating a New Migration

Increment the version number and create both up and down files:

```bash
touch 000007_your_migration_name.up.sql
touch 000007_your_migration_name.down.sql
```

The down file is needed for reversibility.  See [golang-migrate docs: Reversibility of Migrations](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md#reversibility-of-migrations).

## Running Migrations Manually

Migrations run automatically on server startup. Manual commands are typically only needed for rollbacks or testing.

If you need to migrate manually, you'll want to use the CLI.  Installation instructions and examples at [migrate/cmd/migrate/README.md](https://github.com/golang-migrate/migrate/blob/257fa847d614efe3948c25e9033e92b930527dec/cmd/migrate/README.md).
