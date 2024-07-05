package value

type Value interface {
	Dig(key string) Value
	IsTrue() bool
	IsNull() bool
	Error() error
	ToString() (string, error)
}
