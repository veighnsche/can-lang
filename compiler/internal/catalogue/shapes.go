package catalogue

// RuntimeShape is generated from the same closed inventory as declaration IDs.
// Its descriptors retain template parameters; no TypeScript parser is needed.
type RuntimeShape struct {
	Name       string         `json:"name"`
	Identity   string         `json:"identity"`
	Kind       string         `json:"kind"`
	Parameters []Parameter    `json:"parameters"`
	Fields     []RuntimeField `json:"fields"`
	Leaves     []Descriptor   `json:"leaves"`
}
type RuntimeField struct {
	Name string     `json:"name"`
	Type Descriptor `json:"type"`
}

func (c *Catalogue) runtimeShapes() ([]RuntimeShape, error) {
	var out []RuntimeShape
	add := func(name, identity, kind string, parameters []Parameter, fields []Field, leaves []string) error {
		shape := RuntimeShape{Name: name, Identity: identity, Kind: kind, Parameters: parameters, Fields: []RuntimeField{}, Leaves: []Descriptor{}}
		for _, f := range fields {
			d, err := ParseDescriptor(f.Type)
			if err != nil {
				return err
			}
			shape.Fields = append(shape.Fields, RuntimeField{f.Name, d})
		}
		for _, leaf := range leaves {
			d, err := ParseDescriptor(leaf)
			if err != nil {
				return err
			}
			shape.Leaves = append(shape.Leaves, d)
		}
		out = append(out, shape)
		return nil
	}
	for _, d := range c.inventory.Types {
		if err := add(d.Name, d.Identity, d.Kind, d.Parameters, d.Fields, d.Leaves); err != nil {
			return nil, err
		}
	}
	for _, d := range c.inventory.Errors {
		if err := add(d.Name, d.Identity, "error", d.Parameters, d.Fields, nil); err != nil {
			return nil, err
		}
	}
	return out, nil
}
