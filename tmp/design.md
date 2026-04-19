# SYNTAX

- elements separated by whitespace
- define blocks with {} -- whitespace is irrelevant
- can define components on top of file with $; use them with $
- comments start with #
- custom strings inside of ""
- built in helpers (whitespace, anyChar, etc) without quotes
- can add quantifiers to elements with numeric ranges only
- built in patterns start with @

## Quantifiers
1 or more
    a 1..
x or more
    a x.. 
between x and y
    a x..y

```
require customVar             # --customVar must be provided by user (a literal string)
$id = letter digit 1..
$id = letter digit 2..4

pattern:
$customVar
"<%"
whitespace 1..

"a" 1..5
"b" 3    # {3}
"c" 4..
"d" 1..
"e" 1
"f" 1..

$id
capture whateverName { anyChar 1.. }

or (
     ( "a" whitespace ) 
     ( "b" whitespace ) 
     ( "c" ) 
)

(
    @email
    "a"
) 1..2

whitespace 1..
@email                           # built in
['$#@' digit newline]              # enforcing '' instead of "" to show that it matches chars individually (as runes) 
"%>"

```

# USAGE

## As a cli tool

pattern = {-f file.lx | "inline"}
content = "inline"

lx 
    find
    findall
    test
        <pattern> <content> [--name=value ...]
    replace <pattern> "replacement" <content> [--name=value ...]













