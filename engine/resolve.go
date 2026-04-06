package engine

import "fmt"

type resolver struct {
	ast             program
	cliVars         map[string]string
	definitions     map[string]expr
	definitionOrder []defineExpr
	resolving       map[string]struct{}
	resolvedVars    map[string]expr
	usedVars        map[string]struct{}
	usedCaptures    map[string]struct{}
	usedBuiltins    map[string]struct{}
}

func newResolver(ast program, cliVars map[string]string) *resolver {
	return &resolver{
		ast:             ast,
		cliVars:         cliVars,
		definitions:     make(map[string]expr),
		definitionOrder: nil,
		resolving:       make(map[string]struct{}),
		resolvedVars:    make(map[string]expr),
		usedVars:        make(map[string]struct{}),
		usedCaptures:    make(map[string]struct{}),
		usedBuiltins:    make(map[string]struct{}),
	}
}

func (resolver *resolver) resolve() (program, *lxError) {
	var expressions []expr

	for _, expression := range resolver.ast.expressions {
		seenPatternSection := false

		switch current := expression.(type) {
		case defineExpr:
			if seenPatternSection {
				return program{}, &lxError{
					msg: "definitions are not allowed after 'pattern:': $" + current.name,
					pos: current.pos,
				}
			}
			if _, exists := resolver.definitions[current.name]; exists {
				return program{}, &lxError{
					msg: "duplicate variable definition: $" + current.name,
					pos: current.pos,
				}
			}
			resolver.definitions[current.name] = current.value
			resolver.definitionOrder = append(resolver.definitionOrder, current)
		case patternSectionExpr:
			seenPatternSection = true
		default:
			resolvedExpr, err := resolver.resolveExpr(current)
			if err != nil {
				return program{}, err
			}
			expressions = appendNormalizedResolvedExpr(expressions, resolvedExpr)
		}
	}

	for _, definition := range resolver.definitionOrder {
		if _, ok := resolver.usedVars[definition.name]; ok {
			continue
		}
		return program{}, &lxError{
			msg: "unused variable: $" + definition.name,
			pos: definition.pos,
		}
	}

	return program{expressions: expressions}, nil
}

func (resolver *resolver) resolveExpr(expression expr) (expr, *lxError) {
	switch current := expression.(type) {
	case identExpr:
		return resolver.resolveIdent(current)
	case variableExpr:
		return resolver.resolveVar(current)
	case requiredVarExpr:
		return resolver.resolveRequiredVar(current)
	case notExpr:
		target, err := resolver.resolveExpr(current.target)
		if err != nil {
			return nil, err
		}
		return notExpr{pos: current.pos, target: target}, nil
	case builtinExpr:
		if _, ok := resolver.usedBuiltins[current.name]; ok {
			return nil, &lxError{
				msg: "cannot use a builtin multiple times: @" + current.name,
				pos: current.pos,
			}
		}
		resolver.usedBuiltins[current.name] = struct{}{}
		return current, nil
	case literalValueExpr:
		return current, nil
	case classExpr:
		return current, nil
	case groupExpr:
		group, err := resolver.resolveGroup(current)
		if err != nil {
			return nil, err
		}
		return unwrapRedundantResolvedGroup(group), nil
	case captureExpr:
		if _, ok := resolver.usedCaptures[current.name]; ok {
			return nil, &lxError{
				msg: "duplicate capture name: " + current.name,
				pos: current.pos,
			}
		}
		resolver.usedCaptures[current.name] = struct{}{}

		group, err := resolver.resolveGroup(current.group)
		if err != nil {
			return nil, err
		}
		if isResolvedEmptyExpr(group) {
			return nil, &lxError{
				msg: "capture group cannot be an empty string literal",
				pos: current.pos,
			}
		}
		return captureExpr{pos: current.pos, name: current.name, group: group}, nil
	case quantifierExpr:
		target, err := resolver.resolveExpr(current.target)
		if err != nil {
			return nil, err
		}
		if isResolvedEmptyExpr(target) {
			return nil, &lxError{
				msg: "empty string literals cannot be quantified",
				pos: current.pos,
			}
		}
		return quantifierExpr{pos: current.pos, target: target, min: current.min, max: current.max}, nil
	case defineExpr:
		return nil, &lxError{
			msg: fmt.Sprintf("unexpected definition of $%s during resolution", current.name),
			pos: current.pos,
		}
	case patternSectionExpr:
		return nil, &lxError{
			msg: "unexpected pattern section during resolution",
			pos: current.pos,
		}
	default:
		return nil, &lxError{
			msg: "unknown expression during resolution",
			pos: expression.exprPos(),
		}
	}
}

func (resolver *resolver) resolveIdent(ident identExpr) (expr, *lxError) {
	if _, ok := runeIdents[ident.name]; ok {
		return ident, nil
	}

	if _, ok := wildcardIdents[ident.name]; ok {
		return ident, nil
	}

	return nil, &lxError{
		msg: "unknown identifier: " + ident.name,
		pos: ident.pos,
	}
}

func (resolver *resolver) resolveGroup(group groupExpr) (groupExpr, *lxError) {
	branches := make([][]expr, 0, len(group.branches))

	for _, branch := range group.branches {
		resolvedBranch := make([]expr, 0, len(branch))
		for _, expression := range branch {
			resolvedExpr, err := resolver.resolveExpr(expression)
			if err != nil {
				return groupExpr{}, err
			}
			resolvedBranch = appendNormalizedResolvedExpr(resolvedBranch, resolvedExpr)
		}
		branches = append(branches, resolvedBranch)
	}

	return groupExpr{pos: group.pos, branches: branches}, nil
}

func (resolver *resolver) resolveVar(variable variableExpr) (expr, *lxError) {
	resolver.usedVars[variable.name] = struct{}{}

	if resolved, ok := resolver.resolvedVars[variable.name]; ok {
		return resolved, nil
	}

	if _, ok := resolver.resolving[variable.name]; ok {
		return nil, &lxError{
			msg: "cyclic variable definition: $" + variable.name,
			pos: variable.pos,
		}
	}

	definition, ok := resolver.definitions[variable.name]
	if !ok {
		return nil, &lxError{
			msg: "undefined variable: $" + variable.name,
			pos: variable.pos,
		}
	}

	resolver.resolving[variable.name] = struct{}{}
	resolved, err := resolver.resolveExpr(definition)
	delete(resolver.resolving, variable.name)
	if err != nil {
		return nil, err
	}

	resolver.resolvedVars[variable.name] = resolved
	return resolved, nil
}

func (resolver *resolver) resolveRequiredVar(required requiredVarExpr) (expr, *lxError) {
	value, ok := resolver.cliVars[required.name]
	if !ok {
		return nil, &lxError{
			msg: fmt.Sprintf("missing required variable: %s\nHint: add --%s=... to your command", required.name, required.name),
			pos: required.pos,
		}
	}

	return literalValueExpr{pos: required.pos, value: value}, nil
}

func isResolvedEmptyExpr(expression expr) bool {
	if expression == nil {
		return true
	}

	switch current := expression.(type) {
	case literalValueExpr:
		return current.value == ""
	case groupExpr:
		if len(current.branches) != 1 {
			return false
		}
		for _, branchExpr := range current.branches[0] {
			if !isResolvedEmptyExpr(branchExpr) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func appendNormalizedResolvedExpr(expressions []expr, expression expr) []expr {
	if isResolvedEmptyExpr(expression) {
		return expressions
	}

	return append(expressions, expression)
}

func unwrapRedundantResolvedGroup(group groupExpr) expr {
	if len(group.branches) != 1 || len(group.branches[0]) != 1 {
		return group
	}

	nestedGroup, ok := group.branches[0][0].(groupExpr)
	if !ok {
		return group
	}

	return nestedGroup
}
