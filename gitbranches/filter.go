package gitbranches

type Filter func(*Branch) bool

func ParseFilter(specs []string) Filter {
	return func(b *Branch) bool {
		for _, spec := range specs {
			if f, ok := filterFuncs[spec]; ok {
				if !f(b) {
					return false
				}
			}
		}
		return true
	}
}

var (
	filterFuncs = map[string]Filter{
		"current": func(b *Branch) bool {
			return b.Current
		},
		"!current": func(b *Branch) bool {
			return !b.Current
		},
		"living": func(b *Branch) bool {
			return b.Living
		},
		"!living": func(b *Branch) bool {
			return !b.Living
		},
		"upstream": func(b *Branch) bool {
			return b.Upstream != ""
		},
		"!upstream": func(b *Branch) bool {
			return b.Upstream == ""
		},
	}

	filterNames = []string{
		"current",
		"!current",
		"living",
		"!living",
		"upstream",
		"!upstream",
	}
)
