package index

import (
	"lamina/internal/catalog"
	"lamina/internal/versions"
)

type Registry struct {
	Trees   map[string]*Tree
	Reverse map[string]*Reverse
}

func Build(c *catalog.Catalog, s *versions.Store) *Registry {
	r := &Registry{Trees: map[string]*Tree{}, Reverse: map[string]*Reverse{}}
	for name, definition := range c.Indexes {
		tree := New()
		for key, chain := range s.Tables[definition.Relation] {
			for _, v := range chain {
				if !v.Deleted {
					tree.Add(v.Row[definition.Column], key)
				}
			}
		}
		r.Trees[name] = tree
	}
	for name, definition := range c.References {
		rev := NewReverse()
		for key, chain := range s.Tables[definition.Child] {
			for _, v := range chain {
				if !v.Deleted {
					rev.Add(v.Row[definition.Column], key)
				}
			}
		}
		r.Reverse[name] = rev
	}
	return r
}
func (r *Registry) Apply(c *catalog.Catalog, writes []versions.Mutation) {
	for _, m := range writes {
		if m.Deleted {
			continue
		}
		for name, i := range c.Indexes {
			if i.Relation == m.Relation {
				r.Trees[name].Add(m.Row[i.Column], m.Key)
			}
		}
		for name, definition := range c.References {
			if definition.Child == m.Relation {
				r.Reverse[name].Add(m.Row[definition.Column], m.Key)
			}
		}
	}
}
