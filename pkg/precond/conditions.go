package precond

func MustNotBeNil(value any, message string) {
	if value == nil {
		panic(message)
	}
}
