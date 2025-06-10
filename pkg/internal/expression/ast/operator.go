package ast

type Op string

const (
	Nop              = Op("nop")
	Or               = Op("||")
	And              = Op("&&")
	Equal            = Op("==")
	NotEqual         = Op("!=")
	LessThan         = Op("<")
	LessThanEqual    = Op("<=")
	GreaterThan      = Op(">")
	GreaterThanEqual = Op(">=")
	Add              = Op("+")
	Subtract         = Op("-")
	Multiply         = Op("*")
	Divide           = Op("/")
	Not              = Op("!")
)

func (o Op) Precedence() int {
	switch o {
	case Or:
		return 1
	case And:
		return 2
	case Equal, NotEqual,
		LessThan, LessThanEqual,
		GreaterThan, GreaterThanEqual:
		return 3
	case Add, Subtract:
		return 4
	case Multiply, Divide:
		return 5
	}
	return 0
}
