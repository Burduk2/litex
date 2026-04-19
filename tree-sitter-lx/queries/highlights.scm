(comment) @comment

(pattern_section) @keyword
(required_variable) @keyword
(negation "not" @keyword)

(variable) @variable
(builtin) @function.builtin
(identifier) @variable

(capture
  name: (identifier) @type)

(definition
  name: (variable) @variable.parameter)

(number) @number
(string) @string
(character_class) @string.special

[
  "{"
  "}"
  "("
  ")"
  "="
  ":"
  "|"
  ".."
] @punctuation.delimiter
