package engine

import "fmt"

type resolver struct {
	ast program

	cliVars map[string]string

	definitions  map[string]expr
	resolving    map[string]struct{}
	resolvedVars map[string]expr
	usedBuiltins map[string]struct{}
}

func newResolver(ast program, cliVars map[string]string) *resolver {
	return &resolver{
		ast:          ast,
		cliVars:      cliVars,
		definitions:  make(map[string]expr),
		resolving:    make(map[string]struct{}),
		resolvedVars: make(map[string]expr),
		usedBuiltins: make(map[string]struct{}),
	}
}

func (resolver *resolver) resolve() (program, *lxError) {
	var expressions []expr

	for _, expression := range resolver.ast.expressions {
		seenPatternSection := false

		switch current := expression.(type) {
		case defineExpr:
			if seenPatternSection {
				return program{}, &lxError{msg: "definitions are not allowed after 'pattern:': $" + current.name}
			}
			if _, exists := resolver.definitions[current.name]; exists {
				return program{}, &lxError{
					msg: "duplicate variable definition: $" + current.name,
				}
			}
			resolver.definitions[current.name] = current.value
		case patternSectionExpr:
			seenPatternSection = true
		default:
			resolvedExpr, err := resolver.resolveExpr(current)
			if err != nil {
				return program{}, err
			}
			expressions = append(expressions, resolvedExpr)
		}
	}

	return program{expressions: expressions}, nil
}

func (resolver *resolver) resolveExpr(expression expr) (expr, *lxError) {
	switch current := expression.(type) {
	case identExpr:
		return current, nil
	case variableExpr:
		return resolver.resolveVar(current.name)
	case requiredVarExpr:
		return resolver.resolveRequiredVar(current)
	case notExpr:
		target, err := resolver.resolveExpr(current.target)
		if err != nil {
			return nil, err
		}
		return notExpr{target: target}, nil
	case builtinExpr:
		if _, ok := resolver.usedBuiltins[current.name]; ok {
			return nil, &lxError{msg: "cannot use a builtin multiple times: @" + current.name}
		}
		resolver.usedBuiltins[current.name] = struct{}{}
		return current, nil
	case literalValueExpr:
		return current, nil
	case classExpr:
		return current, nil
	case groupExpr:
		return resolver.resolveGroup(current)
	case captureExpr:
		group, err := resolver.resolveGroup(current.group)
		if err != nil {
			return nil, err
		}
		return captureExpr{name: current.name, group: group}, nil
	case quantifierExpr:
		target, err := resolver.resolveExpr(current.target)
		if err != nil {
			return nil, err
		}
		return quantifierExpr{target: target, min: current.min, max: current.max}, nil
	case defineExpr:
		return nil, &lxError{msg: fmt.Sprintf("unexpected definition of $%s during resolution", current.name)}
	case patternSectionExpr:
		return nil, &lxError{msg: "unexpected pattern section during resolution"}
	default:
		return nil, &lxError{msg: "unknown expression during resolution"}
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
			resolvedBranch = append(resolvedBranch, resolvedExpr)
		}
		branches = append(branches, resolvedBranch)
	}

	return groupExpr{branches: branches}, nil
}

func (resolver *resolver) resolveVar(name string) (expr, *lxError) {
	if resolved, ok := resolver.resolvedVars[name]; ok {
		return resolved, nil
	}

	if _, ok := resolver.resolving[name]; ok {
		return nil, &lxError{
			msg: "cyclic variable definition: $" + name,
		}
	}

	definition, ok := resolver.definitions[name]
	if !ok {
		return nil, &lxError{
			msg: "undefined variable: $" + name,
		}
	}

	resolver.resolving[name] = struct{}{}
	resolved, err := resolver.resolveExpr(definition)
	delete(resolver.resolving, name)
	if err != nil {
		return nil, err
	}

	resolver.resolvedVars[name] = resolved
	return resolved, nil
}

func (resolver *resolver) resolveRequiredVar(required requiredVarExpr) (expr, *lxError) {
	value, ok := resolver.cliVars[required.name]
	if !ok {
		return nil, &lxError{msg: "missing required variable: " + required.name}
	}

	return literalValueExpr{value: value}, nil
}
