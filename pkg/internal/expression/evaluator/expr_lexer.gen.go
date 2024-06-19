
//line expr_lexer.rl:1
package evaluator

import (
  "strconv"
  "strings"
)


//line expr_lexer.gen.go:12
const expr_start int = 7
const expr_first_final int = 7
const expr_error int = 0

const expr_en_main int = 7


//line expr_lexer.rl:14


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
  
//line expr_lexer.gen.go:52
	{
	 lex.cs = expr_start
	 lex.ts = 0
	 lex.te = 0
	 lex.act = 0
	}

//line expr_lexer.rl:45
  return lex
}

func (lex *expressionParser) Lex(out *exprSymType) int {
  eof := lex.pe
  tok := 0
  
//line expr_lexer.gen.go:68
	{
	if ( lex.p) == ( lex.pe) {
		goto _test_eof
	}
	switch  lex.cs {
	case 7:
		goto st_case_7
	case 0:
		goto st_case_0
	case 1:
		goto st_case_1
	case 2:
		goto st_case_2
	case 3:
		goto st_case_3
	case 4:
		goto st_case_4
	case 8:
		goto st_case_8
	case 5:
		goto st_case_5
	case 9:
		goto st_case_9
	case 10:
		goto st_case_10
	case 6:
		goto st_case_6
	}
	goto st_out
tr0:
//line expr_lexer.rl:75
 lex.te = ( lex.p)+1
{ tok = NOT_EQUAL; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr3:
//line expr_lexer.rl:66
 lex.te = ( lex.p)+1
{ tok = STRING; out.str = lex.unquoted2(1, 1); {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr5:
//line expr_lexer.rl:68
 lex.te = ( lex.p)+1
{ tok = AND; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr6:
//line expr_lexer.rl:74
 lex.te = ( lex.p)+1
{ tok = EQUAL; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr7:
//line expr_lexer.rl:69
 lex.te = ( lex.p)+1
{ tok = OR; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr8:
//line expr_lexer.rl:80
 lex.te = ( lex.p)+1

	goto st7
tr11:
//line expr_lexer.rl:70
 lex.te = ( lex.p)+1
{ tok = OPEN; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr12:
//line expr_lexer.rl:71
 lex.te = ( lex.p)+1
{ tok = CLOSE; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr13:
//line expr_lexer.rl:72
 lex.te = ( lex.p)+1
{ tok = SEPARATOR; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr14:
//line expr_lexer.rl:73
 lex.te = ( lex.p)+1
{ tok = DOT; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr16:
//line expr_lexer.rl:77
 lex.te = ( lex.p)+1
{ tok = COLON; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr21:
//line expr_lexer.rl:59
 lex.te = ( lex.p)
( lex.p)--
{ out.number, _ = strconv.ParseInt(lex.str(), 10, 64); tok = NUMBER; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr22:
//line expr_lexer.rl:76
 lex.te = ( lex.p)
( lex.p)--
{ tok = CONDITION; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr23:
//line expr_lexer.rl:78
 lex.te = ( lex.p)+1
{ tok = COALESCE; {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
tr24:
//line expr_lexer.rl:62
 lex.te = ( lex.p)
( lex.p)--
{ tok = ID; out.id = lex.str(); {( lex.p)++;  lex.cs = 7; goto _out } }
	goto st7
	st7:
//line NONE:1
 lex.ts = 0

		if ( lex.p)++; ( lex.p) == ( lex.pe) {
			goto _test_eof7
		}
	st_case_7:
//line NONE:1
 lex.ts = ( lex.p)

//line expr_lexer.gen.go:187
		switch  lex.data[( lex.p)] {
		case 32:
			goto tr8
		case 33:
			goto st1
		case 34:
			goto st2
		case 38:
			goto st4
		case 40:
			goto tr11
		case 41:
			goto tr12
		case 44:
			goto tr13
		case 46:
			goto tr14
		case 58:
			goto tr16
		case 61:
			goto st5
		case 63:
			goto st9
		case 95:
			goto st10
		case 124:
			goto st6
		}
		switch {
		case  lex.data[( lex.p)] < 48:
			if 9 <=  lex.data[( lex.p)] &&  lex.data[( lex.p)] <= 13 {
				goto tr8
			}
		case  lex.data[( lex.p)] > 57:
			switch {
			case  lex.data[( lex.p)] > 90:
				if 97 <=  lex.data[( lex.p)] &&  lex.data[( lex.p)] <= 122 {
					goto st10
				}
			case  lex.data[( lex.p)] >= 65:
				goto st10
			}
		default:
			goto st8
		}
		goto st0
st_case_0:
	st0:
		 lex.cs = 0
		goto _out
	st1:
		if ( lex.p)++; ( lex.p) == ( lex.pe) {
			goto _test_eof1
		}
	st_case_1:
		if  lex.data[( lex.p)] == 61 {
			goto tr0
		}
		goto st0
	st2:
		if ( lex.p)++; ( lex.p) == ( lex.pe) {
			goto _test_eof2
		}
	st_case_2:
		switch  lex.data[( lex.p)] {
		case 34:
			goto tr3
		case 92:
			goto st3
		}
		goto st2
	st3:
		if ( lex.p)++; ( lex.p) == ( lex.pe) {
			goto _test_eof3
		}
	st_case_3:
		goto st2
	st4:
		if ( lex.p)++; ( lex.p) == ( lex.pe) {
			goto _test_eof4
		}
	st_case_4:
		if  lex.data[( lex.p)] == 38 {
			goto tr5
		}
		goto st0
	st8:
		if ( lex.p)++; ( lex.p) == ( lex.pe) {
			goto _test_eof8
		}
	st_case_8:
		if 48 <=  lex.data[( lex.p)] &&  lex.data[( lex.p)] <= 57 {
			goto st8
		}
		goto tr21
	st5:
		if ( lex.p)++; ( lex.p) == ( lex.pe) {
			goto _test_eof5
		}
	st_case_5:
		if  lex.data[( lex.p)] == 61 {
			goto tr6
		}
		goto st0
	st9:
		if ( lex.p)++; ( lex.p) == ( lex.pe) {
			goto _test_eof9
		}
	st_case_9:
		if  lex.data[( lex.p)] == 58 {
			goto tr23
		}
		goto tr22
	st10:
		if ( lex.p)++; ( lex.p) == ( lex.pe) {
			goto _test_eof10
		}
	st_case_10:
		if  lex.data[( lex.p)] == 95 {
			goto st10
		}
		switch {
		case  lex.data[( lex.p)] < 65:
			if 48 <=  lex.data[( lex.p)] &&  lex.data[( lex.p)] <= 57 {
				goto st10
			}
		case  lex.data[( lex.p)] > 90:
			if 97 <=  lex.data[( lex.p)] &&  lex.data[( lex.p)] <= 122 {
				goto st10
			}
		default:
			goto st10
		}
		goto tr24
	st6:
		if ( lex.p)++; ( lex.p) == ( lex.pe) {
			goto _test_eof6
		}
	st_case_6:
		if  lex.data[( lex.p)] == 124 {
			goto tr7
		}
		goto st0
	st_out:
	_test_eof7:  lex.cs = 7; goto _test_eof
	_test_eof1:  lex.cs = 1; goto _test_eof
	_test_eof2:  lex.cs = 2; goto _test_eof
	_test_eof3:  lex.cs = 3; goto _test_eof
	_test_eof4:  lex.cs = 4; goto _test_eof
	_test_eof8:  lex.cs = 8; goto _test_eof
	_test_eof5:  lex.cs = 5; goto _test_eof
	_test_eof9:  lex.cs = 9; goto _test_eof
	_test_eof10:  lex.cs = 10; goto _test_eof
	_test_eof6:  lex.cs = 6; goto _test_eof

	_test_eof: {}
	if ( lex.p) == eof {
		switch  lex.cs {
		case 8:
			goto tr21
		case 9:
			goto tr22
		case 10:
			goto tr24
		}
	}

	_out: {}
	}

//line expr_lexer.rl:84


  return tok;
}

func (lex *expressionParser) Error(e string) {
  lex.errors = append(lex.errors, e)
}
