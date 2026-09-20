package github

type ActionProvider struct{}

func (p ActionProvider) Check(commitSHA string) (bool, error) {
	return false, nil
}
