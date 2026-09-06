package git

type Provider struct{}

func (p Provider) Collect() string {
	return "Git evidence"
}
