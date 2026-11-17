package catalog

import (
	"fmt"
	"sort"
)

type Index struct {
	Name     string `json:"name"`
	Relation string `json:"relation"`
	Column   string `json:"column"`
	Unique   bool   `json:"unique"`
	Created  uint64 `json:"created"`
}

// Reference is a cross-relation constraint: every non-null value of
// Column in Child must match the primary key of a visible Parent row.
// Enforcement is deferred to commit and validated against the final
// committed state; deleting a still-referenced parent row is rejected.
type Reference struct {
	Name    string `json:"name"`
	Child   string `json:"child"`
	Column  string `json:"column"`
	Parent  string `json:"parent"`
	Created uint64 `json:"created"`
}
type Catalog struct {
	Relations  map[string]Relation  `json:"relations"`
	Indexes    map[string]Index     `json:"indexes"`
	References map[string]Reference `json:"references"`
}

func New() *Catalog {
	return &Catalog{Relations: map[string]Relation{}, Indexes: map[string]Index{}, References: map[string]Reference{}}
}
func (c *Catalog) Relation(name string) (Relation, error) {
	r, ok := c.Relations[name]
	if !ok {
		return r, fmt.Errorf("unknown relation %s", name)
	}
	return r, nil
}
func (c *Catalog) CheckIndex(i Index) error {
	if !ValidName(i.Name) {
		return fmt.Errorf("invalid index name")
	}
	if _, exists := c.Indexes[i.Name]; exists {
		return fmt.Errorf("index already exists")
	}
	r, err := c.Relation(i.Relation)
	if err != nil {
		return err
	}
	if _, ok := r.Column(i.Column); !ok {
		return fmt.Errorf("unknown index column")
	}
	return nil
}
func (c *Catalog) CheckReference(def Reference) error {
	if !ValidName(def.Name) {
		return fmt.Errorf("invalid reference name")
	}
	if _, exists := c.References[def.Name]; exists {
		return fmt.Errorf("reference already exists")
	}
	child, err := c.Relation(def.Child)
	if err != nil {
		return err
	}
	parent, err := c.Relation(def.Parent)
	if err != nil {
		return err
	}
	column, ok := child.Column(def.Column)
	if !ok {
		return fmt.Errorf("unknown reference column")
	}
	primary, _ := parent.Column(parent.Primary)
	if column.Type != primary.Type {
		return fmt.Errorf("reference column type %s does not match %s primary type %s", column.Type, parent.Name, primary.Type)
	}
	return nil
}
func (c *Catalog) List() []Relation {
	out := make([]Relation, 0, len(c.Relations))
	for _, r := range c.Relations {
		r.Columns = append([]Column(nil), r.Columns...)
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (c *Catalog) IndexList() []Index {
	out := make([]Index, 0, len(c.Indexes))
	for _, v := range c.Indexes {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (c *Catalog) ReferenceList() []Reference {
	out := make([]Reference, 0, len(c.References))
	for _, v := range c.References {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (c *Catalog) Clone() *Catalog {
	n := New()
	for _, r := range c.List() {
		n.Relations[r.Name] = r
	}
	for k, v := range c.Indexes {
		n.Indexes[k] = v
	}
	for k, v := range c.References {
		n.References[k] = v
	}
	return n
}
