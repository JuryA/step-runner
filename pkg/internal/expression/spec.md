# Expression Language Specification

- [Introduction](#introduction)
- [Notation](#notation)
- [Source Code Representation](#source-code-representation)
  - [Characters](#characters)
  - [Letters and Digits](#letters-and-digits)
- [Lexical Elements](#lexical-elements)
  - [Comments](#comments)
  - [Tokens](#tokens)
  - [Identifiers](#identifiers)
  - [Keywords](#keywords)
  - [Operators and Punctuation](#operators-and-punctuation)
  - [Integer Literals](#integer-literals)
  - [Floating-point Literals](#floating-point-literals)
  - [Number Literals](#number-literals)
  - [String Literals](#string-literals)
- [Types](#types)
  - [Boolean](#boolean)
  - [Null](#null)
  - [Number](#number)
  - [String](#string)
  - [Array](#array)
  - [Object](#object)
- [Type Operations](#type-operations)
  - [Type Compatibility Table](#type-compatibility-table)
  - [Equality Semantics](#equality-semantics)
  - [Comparison Semantics](#comparison-semantics)
  - [Invalid Operations](#invalid-operations)
  - [Short-Circuit Evaluation](#short-circuit-evaluation)
- [Expressions](#expressions)
  - [Primary Expressions](#primary-expressions)
  - [Array Literals](#array-literals)
  - [Object Literals](#object-literals)
  - [Selectors](#selectors)
  - [Function Calls](#function-calls)
  - [Operators](#operators)
    - [Unary Operators](#unary-operators)
    - [Binary Operators](#binary-operators)
  - [Template Expressions](#template-expressions)
  - [Truthiness](#truthiness)
  - [Operator Precedence](#operator-precedence)
- [Grammar Summary](#grammar-summary)
- [Implementation Notes](#implementation-notes)
  - [Number Precision](#number-precision)
  - [Type Coercion](#type-coercion)
  - [Error Handling](#error-handling)
  - [Example Expressions](#example-expressions)

## Introduction

This document specifies the syntax and semantics of the expression language. The language is designed for evaluating expressions with support for literals, operators, property access, array indexing, and function calls.

The language is a superset of JSON - any valid JSON document is also a valid expression in this language. Beyond JSON's data literals, the language adds:

- Arithmetic and logical operators
- Property access and array indexing
- Function calls
- Computed object keys
- Single-quoted strings
- Trailing commas in arrays and objects
- Template expressions in strings

## Notation

The syntax is specified using Extended Backus-Naur Form (EBNF):

```ebnf
Production  = production_name "=" Expression "." .
Expression  = Term { "|" Term } .
Term        = Factor { Factor } .
Factor      = production_name | token [ "…" token ] | Group | Option | Repetition .
Group       = "(" Expression ")" .
Option      = "[" Expression "]" .
Repetition  = "{" Expression "}" .
```

Productions are expressions constructed from terms and the following operators, in increasing precedence:

```ebnf
|   alternation
()  grouping
[]  option (0 or 1 times)
{}  repetition (0 to n times)
```

## Source Code Representation

Source code is Unicode text encoded in UTF-8. The text is not canonicalized, so a single accented code point is distinct from the same character constructed from combining an accent and a letter.

### Characters

```ebnf
unicode_char   = /* an arbitrary Unicode code point */ .
unicode_letter = /* a Unicode code point categorized as "Letter" */ .
unicode_digit  = /* a Unicode code point categorized as "Number, decimal digit" */ .
```

### Letters and Digits

```ebnf
letter = unicode_letter | "_" .
digit  = "0" … "9" .
```

## Lexical Elements

### Comments

The language does not currently support comments.

### Tokens

Tokens form the vocabulary of the language. There are four classes: identifiers, keywords, operators and punctuation, and literals.

### Identifiers

Identifiers name variables and functions.

```ebnf
identifier = letter { letter | unicode_digit } .
```

Identifiers must not be keywords. Identifiers are case-sensitive: `foo`, `Foo`, and `FOO` are three different identifiers.

Note: Currently, all identifiers must be provided by the execution context. There is no variable declaration syntax within expressions. The resolution of identifiers (how variables and functions are looked up) is implementation-defined and depends on the execution context.

### Keywords

The following keywords are reserved and may not be used as identifiers:

```text
array       as          break       case        const
continue    default     else        fallthrough float
for         func        function    goto        if
import      in          int         let         loop
map         namespace   number      object      package
range       return      string      struct      switch
type        var         void        while
```

Additionally, the following literal keywords are recognized:

```text
false       null        true
```

### Operators and Punctuation

The following character sequences represent operators and punctuation:

```text
+    &&    ==    !=    (    )
-    ||    <     <=    [    ]
*    !     >     >=    {    }
/    .     ,     :
```

### Integer Literals

Integer literals are sequences of digits. Leading zeros are permitted.

```ebnf
int_lit = digit { digit } .
```

### Floating-point Literals

Floating-point literals consist of an integer part, a decimal point, a fractional part, and an optional exponent part.

```ebnf
float_lit = digit { digit } "." digit { digit } [ exponent ] .
exponent  = ( "e" | "E" ) [ "+" | "-" ] digit { digit } .
```

### Number Literals

A number literal is either an integer or a floating-point literal.

```ebnf
number = int_lit | float_lit .
```

### String Literals

String literals represent string constants. There are two forms: single-quoted and double-quoted.

Single-quoted strings support minimal escape sequences:

- `\\` - backslash
- `\'` - single quote
- `\$` - single dollar sign

Double-quoted strings support standard escape sequences:

- `\a` - alert or bell
- `\b` - backspace
- `\f` - form feed
- `\n` - newline
- `\r` - carriage return
- `\t` - horizontal tab
- `\v` - vertical tab
- `\\` - backslash
- `\"` - double quote
- `\$` - single dollar sign

Both string types support template expressions using `${{ }}` syntax. The expression inside must evaluate to a string. See [Template Expressions](#template-expressions) for details.

```ebnf
string_lit         = single_quoted | double_quoted .
single_quoted      = "'" { unicode_char | single_escape | template } "'" .
double_quoted      = `"` { unicode_char | double_escape | template } `"` .
single_escape      = `\` ( `\` | "'" | "$" ) .
double_escape      = `\` ( "a" | "b" | "f" | "n" | "r" | "t" | "v" | `\` | `"` | "$" ) .
template           = "${{" Expression "}}" .
```

Examples:

```js
// Single-quoted strings
'Hello, world!'
'It\'s a beautiful day'  // Escaped single quote
'Path: C:\\Users\\Alice' // Escaped backslashes

// Double-quoted strings
"Hello, world!"
"She said, \"Hello!\""   // Escaped double quotes
"Line 1\nLine 2\nLine 3" // Newline characters
"Name:\tJohn\nAge:\t30"  // Tab and newline
"Alert\a\tBackspace\b"   // Special characters

// Template expressions
"Hello, ${{ name }}!"                                 // Simple variable interpolation
'User: ${{ user.firstName + " " + user.lastName }}'  // Expression in single quotes
"Path: ${{ dir }}/${{ file }}"                       // Multiple templates
```

## Types

The language supports the following types:

### Boolean

Boolean values are represented by the predeclared constants `true` and `false`.

### Null

The null value is represented by the predeclared constant `null`.

### Number

Numbers are high-precision decimal floating-point values with 128 bits of precision.

### String

Strings are immutable sequences of Unicode code points.

### Array

Arrays are ordered sequences of values. Elements can be of any type and types can be mixed within an array.

### Object

Objects are unordered collections of key-value pairs. Keys must be strings (either string literals or expressions that evaluate to strings). Values can be of any type.

## Type Operations

This section describes which operations are valid between different types and their behavior.

### Type Compatibility Table

| Operation | Valid Types | Result Type | Notes |
|-----------|------------|-------------|-------|
| `+` (binary) | number + number | number | Addition |
| `+` (binary) | string + string | string | Concatenation |
| `-` (binary) | number - number | number | Subtraction |
| `*` | number * number | number | Multiplication |
| `/` | number / number | number | Division (error on divide by zero) |
| `+` (unary) | number | number | Identity (returns unchanged) |
| `-` (unary) | number | number | Negation |
| `!` | any | boolean | Logical NOT based on truthiness |
| `==` | any == any | boolean | Equality comparison |
| `!=` | any != any | boolean | Inequality comparison |
| `<` | any < any | boolean | Less than (see comparison semantics) |
| `<=` | any <= any | boolean | Less than or equal |
| `>` | any > any | boolean | Greater than |
| `>=` | any >= any | boolean | Greater than or equal |
| `&&` | any && any | any | Returns first falsy or last value |
| `\|\|` | any \|\| any | any | Returns first truthy or last value |
| `.` | object/array | any | Property/method access |
| `[]` | object[string] | any | Object property access |
| `[]` | array[number] | any | Array element access |
| `()` | function | any | Function call |

### Equality Semantics

The `==` and `!=` operators compare values as follows:

- **null**: Only equals `null`
- **boolean**: Only equals boolean with same value
- **number**: Equals numbers with same numeric value (comparison using 128-bit precision)
- **string**: Equals strings with identical UTF-8 byte sequences
- **array**: Equals arrays with same length and equal elements (deep comparison)
- **object**: Equals objects with same keys and equal values (deep comparison, key order irrelevant)

### Comparison Semantics

The comparison operators `<`, `<=`, `>`, and `>=` can compare any types. When comparing values:

1. **Same type comparisons**:
   - **numbers**: Numeric comparison
   - **strings**: Lexicographic comparison (UTF-8 byte order)
   - **booleans**: `false < true`
   - **arrays**: Lexicographic comparison - compares elements in order; shorter arrays are less than longer arrays with the same prefix
   - **objects**: Deterministic comparison based on keys and values (implementation-defined ordering)
   - **null**: All null values are equal

   Examples:

   ```js
   [1, 2] < [1, 2, 3]          // true (shorter array with same prefix)
   [1, 3] > [1, 2, 3]          // true (second element 3 > 2)
   {"a": 1} < {"a": 1, "b": 2} // true (fewer keys)
   {"b": 1} > {"a": 2}         // true (key "b" > "a")
   {"a": 2} > {"a": 1}         // true (same keys, but value 2 > 1)
   ```

2. **Different type comparisons**:
   When comparing values of different types, types are ordered as follows:

   | Type | Order |
   |------|-------|
   | null | 0 |
   | boolean | 1 |
   | number | 2 |
   | string | 3 |
   | array | 4 |
   | object | 5 |
   | function | 6 |

   For example:

   ```js
   null < true      // true (null has order 0, boolean has order 1)
   42 < "hello"     // true (number has order 2, string has order 3)
   [1,2,3] > "text" // true (array has order 4, string has order 3)
   ```

### Invalid Operations

The following operations result in runtime errors:

- Arithmetic operations (`-`, `*`, `/`) on non-numeric types
- The `+` operator on mixed types (e.g., string + number)
- Array indexing with non-numeric index
- Object property access with non-string key
- Calling a non-function value
- Accessing properties on `null`
- Division by zero
- Unknown identifiers
- Accessing non-existent object properties
- Array index out of bounds

Note: The `||` operator has special handling for property-not-found and index-out-of-bounds errors. See [Logical Operators](#logical-operators) for details.

### Short-Circuit Evaluation

The logical operators `&&` and `||` use short-circuit evaluation:

- `&&`: If left operand is falsy, right operand is not evaluated
- `||`: If left operand is truthy, right operand is not evaluated

## Expressions

### Primary Expressions

Primary expressions are the operands for unary and binary expressions.

```ebnf
PrimaryExpression = Literal | identifier | "(" Expression ")" | Array | Object .
Literal           = "null" | "true" | "false" | string_lit | number .
```

Parentheses can be used to group expressions and override operator precedence:

```js
2 + 3 * 4        // evaluates to 14
(2 + 3) * 4      // evaluates to 20
```

### Array Literals

Array literals construct array values.

```ebnf
Array         = "[" [ ArrayElements ] "]" .
ArrayElements = Expression { "," Expression } [ "," ] .
```

Example:

```js
[1, 2, 3]
["a", 1, true, null]
[1, 2, 3,]  // trailing comma allowed
```

### Object Literals

Object literals construct object values. Keys can be string literals or expressions that evaluate to strings.

```ebnf
Object         = "{" [ ObjectElements ] "}" .
ObjectElements = ObjectElement { "," ObjectElement } [ "," ] .
ObjectElement  = Expression ":" Expression .
```

Example:

```js
{"name": "John", "age": 30}
{"key": value}
{computed_key: value}          // computed_key must evaluate to string
{"prefix" + "_suffix": value}  // expressions that produce strings
{obj.prop: value,}             // trailing comma allowed
```

Note: Object keys must evaluate to strings at runtime. Non-string keys will result in a runtime error.

### Selectors

Selectors access properties or elements of a value.

```ebnf
Selector = "." identifier | "[" Expression "]" .
```

Property access with `.` requires an identifier. Computed property access with `[]` accepts any expression.

Example:

```js
obj.property
obj["property"]
my_array[0]
my_array[index]
```

### Function Calls

Function calls invoke a function with zero or more arguments.

```ebnf
Call = "(" [ Expression { "," Expression } ] ")" .
```

Example:

```js
my_func()
my_func(1, 2, 3)
obj.method()
my_array[0]()  // if my_array[0] contains a function
```

### Operators

#### Unary Operators

Unary operators have the highest precedence.

```ebnf
UnaryExpression = unary_op UnaryExpression | PostfixExpression .
unary_op        = "+" | "-" | "!" .
```

| Operator | Name | Types | Description |
|----------|------|-------|-------------|
| `+` | unary plus | number | numeric identity |
| `-` | unary minus | number | numeric negation |
| `!` | logical NOT | any | logical negation (based on truthiness) |

#### Binary Operators

Binary operators are left-associative and follow standard precedence rules.

| Precedence | Operators | Associativity |
|------------|-----------|---------------|
| 5 | `*` `/` | left |
| 4 | `+` `-` | left |
| 3 | `==` `!=` `<` `<=` `>` `>=` | left |
| 2 | `&&` | left |
| 1 | `\|\|` | left |

##### Arithmetic Operators

| Operator | Name | Types | Result |
|----------|------|-------|--------|
| `+` | addition | number + number | number |
| `+` | concatenation | string + string | string |
| `-` | subtraction | number - number | number |
| `*` | multiplication | number * number | number |
| `/` | division | number / number | number |

Note: Division by zero results in a runtime error. The `+` operator performs addition for numbers and concatenation for strings. No implicit type conversion occurs - `"hello" + 42` is an error.

##### Comparison Operators

| Operator | Name | Types | Result |
|----------|------|-------|--------|
| `==` | equal | any == any | boolean |
| `!=` | not equal | any != any | boolean |
| `<` | less than | any < any | boolean |
| `<=` | less than or equal | any <= any | boolean |
| `>` | greater than | any > any | boolean |
| `>=` | greater than or equal | any >= any | boolean |

Note: Comparison operators can compare values of any type. See [Comparison Semantics](#comparison-semantics) for details on how different types are compared.

##### Logical Operators

| Operator | Name | Description |
|----------|------|-------------|
| `&&` | logical AND | returns right operand if left is truthy, else left |
| `\|\|` | logical OR | returns left operand if truthy, else right |

Note: Logical operators use short-circuit evaluation and return the actual operand value, not a boolean.

**Special `||` behavior**: When the left operand results in a property-not-found or index-out-of-bounds error, `||` treats this as a falsy value and evaluates the right operand instead of propagating the error.

Examples:

```js
"foo" && "bar"     // "bar" (returns right when left is truthy)
null && "bar"      // null (returns left when left is falsy)
"foo" || "bar"     // "foo" (returns left when left is truthy)
false || "default" // "default" (returns right when left is falsy)

// Special || error handling
obj.missing || "default"    // "default" (missing property treated as falsy)
array[999] || "fallback"    // "fallback" (out of bounds treated as falsy)
obj.exists || "default"     // obj.exists value
```

### Template Expressions

Template expressions allow embedding expressions within string literals using the `${{ }}` syntax. Both single and double-quoted strings support templates.

```ebnf
template = "${{" Expression "}}" .
```

The expression inside the template must evaluate to a string at runtime. Non-string values will result in a runtime error. `\${{` can be used to escape template expressions.

Examples:

```js
// Simple variable interpolation
"Hello, ${{ name }}!"                    // "Hello, Alice!"
'Welcome ${{ user }}'                    // "Welcome Bob"

// Expressions with operators
"Full name: ${{ firstName + " " + lastName }}"
"Path: ${{ dir }}/${{ file }}"

// Multiple templates in one string
"${{ greeting }}, ${{ name }}! Today is ${{ day }}."

// Complex expressions
"User: ${{ user.firstName }} (${{ user.role }})"
"Items: ${{ items[0] }}, ${{ items[1] }}"

// Escape template
"Hello, \${{ \"world!\" }}"             // "Hello, ${{ \"world!\" }}"

// Errors - expression must return string
"Count: ${{ 42 }}"                      // Error: number not string
"Total: ${{ price + tax }}"             // Error: number not string
```

### Truthiness

The following values are considered "falsy":

- `false`
- `null`
- `0` (number zero)
- `""` (empty string)
- `[]` (empty array)
- `{}` (empty object)

All other values are considered "truthy".

### Operator Precedence

The precedence of operators is reflected in the grammar. From lowest to highest:

1. `||` (logical OR)
2. `&&` (logical AND)
3. `==`, `!=`, `<`, `<=`, `>`, `>=` (comparison)
4. `+`, `-` (addition, subtraction)
5. `*`, `/` (multiplication, division)
6. `+`, `-`, `!` (unary operators)
7. `.`, `[]`, `()` (postfix operators)

## Grammar Summary

```ebnf
// Lexical elements
unicode_char   = /* an arbitrary Unicode code point */ .
unicode_letter = /* a Unicode code point categorized as "Letter" */ .
unicode_digit  = /* a Unicode code point categorized as "Number, decimal digit" */ .

letter = unicode_letter | "_" .
digit  = "0" … "9" .

// String escape sequences
escaped_single = `\` ( `\` | "'" ) .
escaped_double = `\` ( "a" | "b" | "f" | "n" | "r" | "t" | "v" | `\` | `"` ) .

// Template expressions
template = "${{" Expression "}}" .

// Tokens (lexical rules)
identifier = letter { letter | unicode_digit } . /* except reserved */
int_lit    = digit { digit } .
float_lit  = digit { digit } "." digit { digit } [ exponent ] .
exponent   = ( "e" | "E" ) [ "+" | "-" ] digit { digit } .
number     = int_lit | float_lit .

string_lit     = single_quoted | double_quoted .
single_quoted  = "'" { unicode_char | escaped_single | template } "'" .
double_quoted  = `"` { unicode_char | escaped_double | template } `"` .
string         = string_lit .

// Operators
binary_op = "||" | "&&" | rel_op | add_op | mul_op .
unary_op  = "+" | "-" | "!" .
rel_op    = "==" | "!=" | "<" | "<=" | ">" | ">=" .
add_op    = "+" | "-" .
mul_op    = "*" | "/" .

// Reserved words
reserved = "array" | "as" | "break" | "case" | "const" | "continue" |
    "default" | "else" | "fallthrough" | "float" | "for" | "func" |
    "function" | "goto" | "if" | "import" | "in" | "int" | "let" | "loop" |
    "map" | "namespace" | "number" | "object" | "package" | "range" |
    "return" | "string" | "struct" | "switch" | "type" | "var" | "void" |
    "while" .

// Expression grammar
Expression = OrExpression .

OrExpression = AndExpression { "||" AndExpression } .
AndExpression = ComparisonExpression { "&&" ComparisonExpression } .
ComparisonExpression = AdditiveExpression { rel_op AdditiveExpression } .
AdditiveExpression = MultiplicativeExpression { add_op MultiplicativeExpression } .
MultiplicativeExpression = UnaryExpression { mul_op UnaryExpression } .

UnaryExpression = unary_op UnaryExpression
                | PostfixExpression .

PostfixExpression = PrimaryExpression
                  { "." identifier
                  | "[" Expression "]"
                  | Call
                  } .

PrimaryExpression = Literal
                  | identifier
                  | "(" Expression ")"
                  | Array
                  | Object .

Literal = "null"
        | "true"
        | "false"
        | string
        | number .

Array = "[" [ ArrayElements ] "]" .
ArrayElements = Expression { "," Expression } [ "," ] .

Object = "{" [ ObjectElements ] "}" .
ObjectElements = ObjectElement { "," ObjectElement } [ "," ] .
ObjectElement = Expression ":" Expression .

Call = "(" [ Expression { "," Expression } ] ")" .
```

## Implementation Notes

### Number Precision

All numeric operations use high-precision decimal arithmetic with 128 bits of precision. While this provides significantly more precision than standard 64-bit floating-point numbers, it is not truly arbitrary-precision and some precision loss may occur in extreme cases.

### Type Coercion

The language performs minimal implicit type coercion:

- Logical operators (`!`, `&&`, `||`) evaluate truthiness but return actual values
- Comparison operators always return boolean values
- Arithmetic operators require numeric operands

### Error Handling

Operations on incompatible types (e.g., adding a string to a number) result in runtime errors. The language does not perform automatic type conversion for arithmetic operations.

### Example Expressions

Here are some example expressions demonstrating various language features:

```js
// complex property access and calls
user.language({ 'default': 'en' })
user.permissions["admin"]

// computed object keys
{"prefix_" + type: value}

// chained operations
(x > 0 && x < 10) || x == 100

// mixed arrays and operations
[1, 2, 3][index % 3] * factor

// nested structures
{
  "users": [
    {"name": "Alice", "age": 30},
    {"name": "Bob", "age": 25}
  ],
  "count": 2
}
```
