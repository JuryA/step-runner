package runner

// Describer types know how to describe themselves in a language to be read by humans
type Describer interface {
	Describe() string
}
