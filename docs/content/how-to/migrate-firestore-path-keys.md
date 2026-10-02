# Migrate Firestore path keys

Use this offline migration when upgrading a Firestore installation from the
legacy slash-to-plus document keys. Stop application writers during the
migration. The new server reads only the encoded keys.

The new layout is `links/<domain>/shortLinks/p-<encoded_path>`, where the
escaped source path is encoded as lowercase RFC 4648 Base32 without padding.
The domain is lowercase. `/` is an explicit root link. Path case, literal
plus, repeated slashes, and escaped reserved characters retain their identity;
percent-escape hex digits are normalized to uppercase.

## Prepare

1. Authenticate with Application Default Credentials for the intended project.
2. Run the tests and build the new server image before starting downtime.
3. Apply `google_firestore_field.link_owner_encoded` from
   `deploy/prod/tf/firestore.tf`. This enables owner queries across the
   `shortLinks` collection group. Retain the old `id` index for rollback.
4. Run the default, read-only migration plan from the repository root:

   ```bash
   go run ./tools/migrate-firestore -project YOUR_PROJECT
   ```

The planner uses `from` when present. Otherwise it follows the old listing
interpretation: each `+` means `/`, and relative paths receive a leading slash.
A domain-root record becomes `/`; empty parent documents are retained.
The old encoding cannot reveal whether an internal `+` originally meant a
literal plus. Review the reported `ambiguous_keys` count against this policy.
The migration cannot recover records already overwritten by historical
collisions.

Destinations, owners, and other string fields are preserved, including empty
destinations and missing owners. Identical records at the same canonical
address are consolidated with every original retained in the backup.
Conflicting destinations or owners, invalid source metadata, unsupported
field types, occupied target keys, and oversized IDs stop planning before
any write occurs.

## Apply during downtime

1. Stop ingress to the old server and let in-flight writes finish. For a
   Cloud Run service behind an external Application Load Balancer, temporarily
   set its ingress to `internal` and wait for its request timeout. Ensure
   internal callers are also stopped.
2. Create a private backup directory outside version control:

   ```bash
   mkdir -m 700 .migration-backups
   ```

3. Apply the plan, saving an exclusive backup file first:

   ```bash
   go run ./tools/migrate-firestore -project YOUR_PROJECT \
     -apply -backup .migration-backups/firestore-path-keys.json
   ```

   The command writes and syncs the backup with permissions `0600`, then
   moves each link in a transaction. Each transaction verifies the source
   update time, creates the absent target, and removes its legacy sources.
   It verifies every target's fields and confirms the old keys are absent.

4. Send traffic to the new server and restore ingress. Check existing
   redirects and an authenticated list. A fresh dry run should report zero
   moves. Source keys in the backup allow inspection without printing owner
   IDs or destinations in command output.

If migration stops partway through, keep writers stopped. The completed
moves are atomic; a new dry run plans only the remaining legacy records.
Use a different backup filename for any subsequent application and retain
the first backup.

## Roll back

Keep writers stopped and run:

```bash
go run ./tools/migrate-firestore -project YOUR_PROJECT \
  -rollback .migration-backups/firestore-path-keys.json
```

Rollback restores the original documents and removes the encoded copies in
transactions. It refuses targets whose fields changed after migration and
verifies already-restored entries, so it also works after a partial migration.
Route traffic back to the old server revision before restoring ingress.

The backup contains link destinations and owner identifiers. Keep it private
and retain it until the migration is accepted.
