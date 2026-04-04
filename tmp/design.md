# SYNTAX

- elements separated by whitespace
- define blocks with {} -- whitespace is irrelevant
- can define components on top of file with $; use them with $
- comments start with #
- custom strings inside of ""
- built in helpers (whitespace, anyChar, etc) without quotes
- can add quantifiers to elements (x+, x-y, ?)
- built in patterns start with @

## Quantifiers
0 or more
    a*
1 or more
    a+
0 or 1
    a?
x or more
    a x.. 
    a x-
    a{x,}
between x and y
    a{x,y}
    a x-y
    a x..y

```
require customVar             # --customVar must be provided by user (a literal string)
$id = letter digit 0+
$id = letter digit* 

pattern:
$customVar
"<%"
whitespace 0+

"a" 1-5  # {1,5}
"b" 3    # {3}
"c" 4-   # {4,}
"d"*     # *
"e"?     # ?
"f"+     # +

$id
capture whateverName { anyChar 0+ }

or (
     ( "a" whitespace ) 
     ( "b" whitespace ) 
     ( "c" ) 
)

(
    @email
    "a"
) 1-2

whitespace 0+
@email                           # built in
['$#@' digit newline]              # enforcing '' instead of "" to show that it matches chars individually (as runes) 
"%>"

```

# USAGE

## As a cli tool

pattern = {@file.lx | "inline"}
content = {"inline" | @file}

lx 
    find
    findall
    test
        <pattern> <content> [--name=value ...]
    replace <pattern> "replacement" <content> [--name=value ...]
















