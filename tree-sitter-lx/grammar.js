/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

const PREC = {
  QUANTIFIER: 10,
};

module.exports = grammar({
  name: "lx",

  extras: ($) => [
    /\s/,
    $.comment,
  ],

  rules: {
    source_file: ($) =>
      repeat(
        choice(
          $.definition,
          $.pattern_section,
          $.expression,
        ),
      ),

    comment: () => token(seq("#", /.*/)),

    pattern_section: () => seq("pattern", ":"),

    definition: ($) =>
      prec.right(
        seq(
          field("name", $.variable),
          "=",
          field("value", repeat1($.expression)),
        ),
      ),

    expression: ($) =>
      choice(
        $.quantified_expression,
        $._expression_base,
      ),

    quantified_expression: ($) =>
      prec.left(
        PREC.QUANTIFIER,
        seq(
          field("target", $._expression_base),
          field("quantifier", $.quantifier),
        ),
      ),

    quantifier: ($) =>
      seq(
        field("min", $.number),
        optional(
          seq(
            "..",
            optional(field("max", $.number)),
          ),
        ),
      ),

    _expression_base: ($) =>
      choice(
        $.capture,
        $.group,
        $.required_variable,
        $.negation,
        $.variable,
        $.builtin,
        $.string,
        $.character_class,
        $.identifier,
      ),

    capture: ($) =>
      seq(
        field("name", $.identifier),
        field("body", $.capture_body),
      ),

    capture_body: ($) =>
      seq(
        "{",
        optional("|"),
        field("branch", $.branch),
        repeat(seq("|", field("branch", $.branch))),
        "}",
      ),

    group: ($) =>
      seq(
        "(",
        optional("|"),
        field("branch", $.branch),
        repeat(seq("|", field("branch", $.branch))),
        ")",
      ),

    branch: ($) => repeat1($.expression),

    negation: ($) =>
      seq(
        "not",
        field("target", choice($.identifier, $.character_class)),
      ),

    required_variable: ($) =>
      seq(
        "require",
        field("name", $.identifier),
      ),

    variable: () => /\$[A-Za-z_][A-Za-z0-9_]*/,
    builtin: () => /@[A-Za-z_][A-Za-z0-9_]*/,
    identifier: () => /[A-Za-z_][A-Za-z0-9_]*/,
    number: () => /0|[1-9][0-9]*/,
    string: () => token(seq('"', repeat(choice(/[^"\\\n]+/, /\\./)), '"')),
    character_class: () => token(seq("[", repeat(choice(/[^\\\]\n]+/, /\\./)), "]")),
  },
});
