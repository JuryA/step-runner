package evaluator

//go:generate ragel -Z -G2 -o expr_lexer.gen.go expr_lexer.rl
//go:generate go run golang.org/x/tools/cmd/goyacc -p expr -o expr_parser.gen.go -v expr_parser.gen.output expr_parser.y
