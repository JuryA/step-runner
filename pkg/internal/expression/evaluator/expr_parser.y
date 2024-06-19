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
%token NUMBER EQUAL NOT_EQUAL ID DOT STRING OPEN CLOSE AND OR SEPARATOR

%type <expr> start expression or_expression and_expression comparison_expression value_expression
%type <exprList> expression_list

%%

start: expression {
  exprlex.(*expressionParser).result = $1;
};

expression_list:
    expression_list SEPARATOR expression { $$ = append($1, $3); }
  | expression { $$ = []Node{ $1 }; }

expression: or_expression { $$ = $1; }

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

value_expression:
    ID { $$ = &nodeDig{expr: &nodeContext{}, key: $1}; }
  | NUMBER { $$ = &nodeValue{value: value.ToValue($1)}; }
  | STRING { $$ = &nodeValue{value: value.ToValue($1)}; }
  | OPEN expression CLOSE { $$ = $2; }
  | value_expression DOT ID { $$ = &nodeDig{expr: $1, key: $3}; }
  | ID OPEN expression_list CLOSE { $$ = &nodeCall{expr: &nodeContext{}, method: $1, args: $3}; }
  | ID OPEN CLOSE { $$ = &nodeCall{expr: &nodeContext{}, method: $1}; }
  | value_expression DOT ID OPEN expression_list CLOSE { $$ = &nodeCall{expr: $1, method: $3, args: $5}; }
  | value_expression DOT ID OPEN CLOSE { $$ = &nodeCall{expr: $1, method: $3}; }
