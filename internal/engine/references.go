package engine

import (
	"lamina/internal/catalog"
	"lamina/internal/index"
	"lamina/internal/journal"
)

// CreateReference installs a cross-relation constraint as a single durable
// DDL record. Existing committed data must already satisfy it; the check is
// a one-time full validation, mirroring unique index creation.
func (d *DB) CreateReference(def catalog.Reference) (catalog.Reference, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.usable(); err != nil {
		return def, err
	}
	if err := d.catalog.CheckReference(def); err != nil {
		return def, invalid(err)
	}
	if err := d.checkReferenceData(def, nil); err != nil {
		return def, err
	}
	def.Created = d.sequence + 1
	if err := d.publish(journal.Record{Kind: "reference", Reference: &def}); err != nil {
		return def, err
	}
	d.indexes = index.Build(d.catalog, d.store)
	return def, nil
}

// references validates every constraint whose child or parent relation is
// touched by the committing transaction. It runs under the database lock at
// commit time, so the watermark is the current committed head overlaid with
// the transaction's own write set — never the transaction's begin snapshot.
// Staged writes are the only per-transaction state consulted; no pending
// dependency bookkeeping is retained and nothing is copied wholesale.
func (d *DB) references(writes map[string]map[string]staged) error {
	for _, def := range d.catalog.ReferenceList() {
		stagedChild := writes[def.Child]
		stagedParent := writes[def.Parent]
		if len(stagedChild) == 0 && len(stagedParent) == 0 {
			continue
		}
		// Child side: every staged put must reference a parent key that is
		// visible in the final state.
		for _, m := range stagedChild {
			if m.Mutation.Deleted {
				continue
			}
			value := m.Mutation.Row[def.Column]
			if value.Null() {
				continue
			}
			if !d.parentVisible(def.Parent, value.Key(), writes) {
				return fault("conflict", "reference %s violated: %s value %s has no %s row", def.Name, def.Child, value.Key(), def.Parent)
			}
		}
		// Parent side: a staged delete must not orphan a committed child.
		// Reverse candidates are a historical superset and are re-interpreted
		// against the final state before they can condemn the commit.
		for key, m := range stagedParent {
			if !m.Mutation.Deleted {
				continue
			}
			for _, childKey := range d.indexes.Reverse[def.Name].Candidates(key) {
				if staged, ok := stagedChild[childKey]; ok {
					if !staged.Mutation.Deleted && staged.Mutation.Row[def.Column].Key() == key {
						return fault("conflict", "reference %s violated: %s row %s still referenced", def.Name, def.Parent, key)
					}
					continue
				}
				row, visible := d.store.Read(def.Child, childKey, d.sequence)
				if visible && row[def.Column].Key() == key {
					return fault("conflict", "reference %s violated: %s row %s still referenced", def.Name, def.Parent, key)
				}
			}
		}
	}
	return nil
}

// parentVisible reports whether the referenced key exists in the final
// state: the committing transaction's own staged mutation wins, otherwise
// the committed head answers.
func (d *DB) parentVisible(parent, token string, writes map[string]map[string]staged) bool {
	if m, ok := writes[parent][token]; ok {
		return !m.Mutation.Deleted
	}
	_, visible := d.store.Read(parent, token, d.sequence)
	return visible
}

// checkReferenceData fully validates one constraint against the committed
// head overlaid with an optional write set. Used at DDL admission and at
// open/recovery time, never per commit.
func (d *DB) checkReferenceData(def catalog.Reference, writes map[string]map[string]staged) error {
	rows := d.store.Scan(def.Child, d.sequence)
	overlay(rows, writes, def.Child)
	for _, row := range rows {
		value := row[def.Column]
		if value.Null() {
			continue
		}
		if !d.parentVisible(def.Parent, value.Key(), writes) {
			return fault("conflict", "reference %s violated: %s value %s has no %s row", def.Name, def.Child, value.Key(), def.Parent)
		}
	}
	return nil
}
