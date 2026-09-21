package catalogue

// Descriptor is the closed inventory's named/array type notation. It is not an
// authored-source parser. Parameters retain their inventory names until the
// concrete type builder substitutes already-resolved nodes.
type Descriptor struct {
	Name      string
	Arguments []Descriptor
}

func ParseDescriptor(text string) (Descriptor, error) {
	ref, err := parseType(text)
	if err != nil {
		return Descriptor{}, err
	}
	var convert func(typeRef) Descriptor
	convert = func(r typeRef) Descriptor {
		out := Descriptor{Name: r.name}
		for _, arg := range r.arguments {
			out.Arguments = append(out.Arguments, convert(arg))
		}
		return out
	}
	return convert(ref), nil
}
