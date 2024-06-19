%{
package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
%}

%union {
   id string
   number int64
   str string
   expr Node
   exprList []Node
}

%token <id> ID
%token <number> NUMBER
%token <str> STRING
%token NUMBER EQUAL NOT_EQUAL ID DOT STRING OPEN CLOSE AND OR SEPARATOR CONDITION COLON COALESCE

%type <expr> start expression or_expression and_expression comparison_expression value_expression optional_value_expression_expansion conditional_expression
%type <exprList> expression_list optional_expression_list

%%

start: expression {
  exprlex.(*expressionParser).result = $1;
};

expression_list:
    expression_list SEPARATOR expression { $$ = append($1, $3); }
  | expression { $$ = []Node{ $1 }; }

expression: conditional_expression { $$ = $1; }

conditional_expression:
    or_expression CONDITION or_expression COLON conditional_expression { $$ = &nodeConditional{check: $1, left: $3, right: $5}; }
  | or_expression COALESCE conditional_expression { $$ = &nodeCoalesce{left: $1, right: $3}; }
  | or_expression { $$ = $1; }

or_expression:
    or_expression OR and_expression { $$ = &nodeOr{left: $1, right: $3}; }
  | and_expression { $$ = $1; }

and_expression:
    and_expression AND comparison_expression { $$ = &nodeAnd{left: $1, right: $3}; }
  | comparison_expression { $$ = $1; }

comparison_expression:
    comparison_expression NOT_EQUAL value_expression { $$ = &nodeCompareNotEquals{left: $1, right: $3}; }
  | comparison_expression EQUAL value_expression { $$ = &nodeCompareEquals{left: $1, right: $3}; }
  | value_expression { $$ = $1; }

optional_expression_list:
    expression_list { $$ = $1; }
  | { clear($$); }

optional_value_expression_expansion:
    value_expression DOT { $$ = $1; }
  | { $$ = &nodeContext{}; }

value_expression:
    OPEN expression CLOSE { $$ = $2; }
  | NUMBER { $$ = &nodeValue{value: value.ToValue($1)}; }
  | STRING { $$ = &nodeValue{value: value.ToValue($1)}; }
  | optional_value_expression_expansion ID { $$ = &nodeDig{expr: $1, key: $2}; }
  | optional_value_expression_expansion ID OPEN optional_expression_list CLOSE { $$ = &nodeCall{expr: $1, method: $2, args: $4}; }
