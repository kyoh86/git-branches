package gitbranches

// Branch contains information of a branch
type Branch struct {
	Current   bool
	Remote    string
	Name      string
	Upstream  string
	Living    bool
	Committer string
}

// FullName gets path of the origin and name
func (b Branch) FullName() string {
	if b.Remote == "" {
		return b.Name
	}
	return b.Remote + "/" + b.Name
}
