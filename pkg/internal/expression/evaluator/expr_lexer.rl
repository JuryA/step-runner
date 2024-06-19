package evaluator

import (
  "strconv"
  "strings"
)

%%{ 
    machine expr;
    write data;
    access lex.;
    variable p lex.p;
    variable pe lex.pe;
}%%

type expressionParser struct {
	data        []byte
	p, pe, cs   int
	ts, te, act int
	result      Node
	errors      []string
}

func (l *expressionParser) str2(loff, roff int) string {
  return string(l.data[l.ts+loff:l.te-roff])
}

func (l *expressionParser) str() string {
  return l.str2(0, 0)
}

func (l *expressionParser) unquoted2(loff, roff int) string {
  s := l.str2(loff, roff)
  s = strings.ReplaceAll(s, "\\\"", "\"");
  s = strings.ReplaceAll(s, "\\\\", "\\");
  return s;
}

func newExpressionParser(data []byte) *expressionParser {
  lex := &expressionParser {
    data: data,
    pe: len(data),
  }
  %% write init;
  return lex
}

func (lex *expressionParser) Lex(out *exprSymType) int {
  eof := lex.pe
  tok := 0
  %%{ 
    main := |*
      newline = '\n';
      any_count_line = any | newline;
      alnum_u = alnum | '_';
      alpha_u = alpha | '_';

      # number
      digit+ => { out.number, _ = strconv.ParseInt(lex.str(), 10, 64); tok = NUMBER; fbreak; };

      # identifier
      alpha_u alnum_u* => { tok = ID; out.id = lex.str(); fbreak; };

      # double quotes: this is missing unquote behavior
      dliteralChar = [^"\\] | newline | ( '\\' any_count_line );
      '"' . dliteralChar* . '"' { tok = STRING; out.str = lex.unquoted2(1, 1); fbreak; };

      "&&" => { tok = AND; fbreak; };
      "||" => { tok = OR; fbreak; };
      "(" => { tok = OPEN; fbreak; };
      ")" => { tok = CLOSE; fbreak; };
      "," => { tok = SEPARATOR; fbreak; };
      "\." => { tok = DOT; fbreak; };
      "==" => { tok = EQUAL; fbreak; };
      "!=" => { tok = NOT_EQUAL; fbreak; };
      "?" => { tok = CONDITION; fbreak; };
      ":" => { tok = COLON; fbreak; };
      "?:" => { tok = COALESCE; fbreak; };

      space;
    *|;

    write exec;
  }%%

  return tok;
}

func (lex *expressionParser) Error(e string) {
  lex.errors = append(lex.errors, e)
}
