package parse

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const end_symbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	rulee
	ruleexprOrConditional
	ruleconditionalExpr
	ruleb
	ruleb1
	ruleb2
	ruletimeComparison
	rulenumericalComparison
	rulelogicalComparison
	rulestringComparison
	rulestringExpression
	rulebooleanValue
	rulee1
	rulee2
	rulee3
	rulee4
	rulevalue
	ruleidentifier
	rulefunctionCall
	rulefunctionArgumentList
	rulefunctionArgument
	rulefunctionName
	rulewholeSeries
	rulespecificIdentifier
	rulegeneralIdentifier
	rulethisIdentifier
	rulespecificIdentifierRange
	rulegeneralIdentifierRange
	rulethisIdentifierRange
	rulecategoryIdentifier
	rulestringValue
	ruletimeRange
	ruletimeIndex
	ruleindexComputation
	ruleindexExpr
	ruleindexBegin
	ruleindexEnd
	ruleindexT
	ruleopenIndex
	rulecloseIndex
	ruleif
	rulethen
	ruleelse
	ruletrue
	rulefalse
	ruleequal
	rulenotEqual
	rulegreaterThan
	rulegreaterThanEqual
	rulelessThan
	rulelessThanEqual
	rulenot
	ruleand
	ruleor
	ruleadd
	ruleminus
	rulemultiply
	ruledivide
	rulemodulus
	ruleexponentiation
	ruleopen
	ruleclose
	rulecomma
	rulequote
	rulecolon
	rulesp
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	ruleAction14
	ruleAction15
	ruleAction16
	ruleAction17
	ruleAction18
	ruleAction19
	ruleAction20
	ruleAction21
	ruleAction22
	rulePegText
	ruleAction23
	ruleAction24
	ruleAction25
	ruleAction26
	ruleAction27
	ruleAction28
	ruleAction29
	ruleAction30
	ruleAction31
	ruleAction32
	ruleAction33
	ruleAction34
	ruleAction35
	ruleAction36
	ruleAction37
	ruleAction38
	ruleAction39
	ruleAction40
	ruleAction41
	ruleAction42
	ruleAction43

	rulePre_
	rule_In_
	rule_Suf
)

var rul3s = [...]string{
	"Unknown",
	"e",
	"exprOrConditional",
	"conditionalExpr",
	"b",
	"b1",
	"b2",
	"timeComparison",
	"numericalComparison",
	"logicalComparison",
	"stringComparison",
	"stringExpression",
	"booleanValue",
	"e1",
	"e2",
	"e3",
	"e4",
	"value",
	"identifier",
	"functionCall",
	"functionArgumentList",
	"functionArgument",
	"functionName",
	"wholeSeries",
	"specificIdentifier",
	"generalIdentifier",
	"thisIdentifier",
	"specificIdentifierRange",
	"generalIdentifierRange",
	"thisIdentifierRange",
	"categoryIdentifier",
	"stringValue",
	"timeRange",
	"timeIndex",
	"indexComputation",
	"indexExpr",
	"indexBegin",
	"indexEnd",
	"indexT",
	"openIndex",
	"closeIndex",
	"if",
	"then",
	"else",
	"true",
	"false",
	"equal",
	"notEqual",
	"greaterThan",
	"greaterThanEqual",
	"lessThan",
	"lessThanEqual",
	"not",
	"and",
	"or",
	"add",
	"minus",
	"multiply",
	"divide",
	"modulus",
	"exponentiation",
	"open",
	"close",
	"comma",
	"quote",
	"colon",
	"sp",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",
	"Action15",
	"Action16",
	"Action17",
	"Action18",
	"Action19",
	"Action20",
	"Action21",
	"Action22",
	"PegText",
	"Action23",
	"Action24",
	"Action25",
	"Action26",
	"Action27",
	"Action28",
	"Action29",
	"Action30",
	"Action31",
	"Action32",
	"Action33",
	"Action34",
	"Action35",
	"Action36",
	"Action37",
	"Action38",
	"Action39",
	"Action40",
	"Action41",
	"Action42",
	"Action43",

	"Pre_",
	"_In_",
	"_Suf",
}

type tokenTree interface {
	Print()
	PrintSyntax()
	PrintSyntaxTree(buffer string)
	Add(rule pegRule, begin, end, next uint32, depth int)
	Expand(index int) tokenTree
	Tokens() <-chan token32
	AST() *node32
	Error() []token32
	trim(length int)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(depth int, buffer string) {
	for node != nil {
		for c := 0; c < depth; c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[node.pegRule], strconv.Quote(string(([]rune(buffer)[node.begin:node.end]))))
		if node.up != nil {
			node.up.print(depth+1, buffer)
		}
		node = node.next
	}
}

func (ast *node32) Print(buffer string) {
	ast.print(0, buffer)
}

type element struct {
	node *node32
	down *element
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	pegRule
	begin, end, next uint32
}

func (t *token32) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: uint32(t.begin), end: uint32(t.end), next: uint32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = uint32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens32) PreOrder() (<-chan state32, [][]token32) {
	s, ordered := make(chan state32, 6), t.Order()
	go func() {
		var states [8]state32
		for i, _ := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, uint32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(string(([]rune(buffer)[token.begin:token.end]))))
	}
}

func (t *tokens32) Add(rule pegRule, begin, end, depth uint32, index int) {
	t.tree[index] = token32{pegRule: rule, begin: uint32(begin), end: uint32(end), next: uint32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

/*func (t *tokens16) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2 * len(tree))
		for i, v := range tree {
			expanded[i] = v.getToken32()
		}
		return &tokens32{tree: expanded}
	}
	return nil
}*/

func (t *tokens32) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	return nil
}

type Calculator struct {
	Expression

	Buffer string
	buffer []rune
	rules  [112]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	tokenTree
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer string, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range []rune(buffer) {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p *Calculator
}

func (e *parseError) Error() string {
	tokens, error := e.p.tokenTree.Error(), "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.Buffer, positions)
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf("parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n",
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			/*strconv.Quote(*/ e.p.Buffer[begin:end] /*)*/)
	}

	return error
}

func (p *Calculator) PrintSyntaxTree() {
	p.tokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *Calculator) Highlighter() {
	p.tokenTree.PrintSyntax()
}

func (p *Calculator) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for token := range p.tokenTree.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.AddOperator(TypeThen)
		case ruleAction1:
			p.AddOperator(TypeElse)
		case ruleAction2:
			p.AddOperator(TypeAnd)
		case ruleAction3:
			p.AddOperator(TypeOr)
		case ruleAction4:
			p.AddOperator(TypeNot)
		case ruleAction5:
			p.AddOperator(TypeTimeEqual)
		case ruleAction6:
			p.AddOperator(TypeEqual)
		case ruleAction7:
			p.AddOperator(TypeNotEqual)
		case ruleAction8:
			p.AddOperator(TypeGreaterThan)
		case ruleAction9:
			p.AddOperator(TypeGreaterThanEqual)
		case ruleAction10:
			p.AddOperator(TypeLessThan)
		case ruleAction11:
			p.AddOperator(TypeLessThanEqual)
		case ruleAction12:
			p.AddOperator(TypeLogicalEqual)
		case ruleAction13:
			p.AddOperator(TypeLogicalNotEqual)
		case ruleAction14:
			p.AddOperator(TypeStringEqual)
		case ruleAction15:
			p.AddOperator(TypeStringNotEqual)
		case ruleAction16:
			p.AddOperator(TypeAdd)
		case ruleAction17:
			p.AddOperator(TypeSubtract)
		case ruleAction18:
			p.AddOperator(TypeMultiply)
		case ruleAction19:
			p.AddOperator(TypeDivide)
		case ruleAction20:
			p.AddOperator(TypeModulus)
		case ruleAction21:
			p.AddOperator(TypeExponentiation)
		case ruleAction22:
			p.AddOperator(TypeNegation)
		case ruleAction23:
			p.AddValue(buffer[begin:end])
		case ruleAction24:
			p.AddFunctionCall()
		case ruleAction25:
			p.AddFunctionArgument()
		case ruleAction26:
			p.AddFunctionName(buffer[begin:end])
		case ruleAction27:
			p.AddIdentifierSpecific(buffer[begin:end])
		case ruleAction28:
			p.AddIdentifierGeneral()
		case ruleAction29:
			p.AddIdentifierThis()
		case ruleAction30:
			p.AddIdentifierSpecificRange(buffer[begin:end])
		case ruleAction31:
			p.AddIdentifierGeneralRange()
		case ruleAction32:
			p.AddIdentifierThisRange()
		case ruleAction33:
			p.AddCategoryIdentifier()
		case ruleAction34:
			p.AddStringValue(buffer[begin:end])
		case ruleAction35:
			p.AddIndexOperator(TypeTimeRange)
		case ruleAction36:
			p.AddIndexOperator(TypeAdd)
		case ruleAction37:
			p.AddIndexOperator(TypeSubtract)
		case ruleAction38:
			p.AddIndexOperator(TypeBegin)
		case ruleAction39:
			p.AddIndexOperator(TypeEnd)
		case ruleAction40:
			p.AddIndexOperator(TypeCurrentTime)
		case ruleAction41:
			p.AddIndexValue(buffer[begin:end])
		case ruleAction42:
			p.AddOperator(TypeTrue)
		case ruleAction43:
			p.AddOperator(TypeFalse)

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *Calculator) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != end_symbol {
		p.buffer = append(p.buffer, end_symbol)
	}

	var tree tokenTree = &tokens32{tree: make([]token32, math.MaxInt16)}
	position, depth, tokenIndex, buffer, _rules := uint32(0), uint32(0), 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokenTree = tree
		if matches {
			p.tokenTree.trim(tokenIndex)
			return nil
		}
		return &parseError{p}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule pegRule, begin uint32) {
		if t := tree.Expand(tokenIndex); t != nil {
			tree = t
		}
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
	}

	matchDot := func() bool {
		if buffer[position] != end_symbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 e <- <(sp exprOrConditional !.)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				if !_rules[rulesp]() {
					goto l0
				}
				if !_rules[ruleexprOrConditional]() {
					goto l0
				}
				{
					position2, tokenIndex2, depth2 := position, tokenIndex, depth
					if !matchDot() {
						goto l2
					}
					goto l0
				l2:
					position, tokenIndex, depth = position2, tokenIndex2, depth2
				}
				depth--
				add(rulee, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 exprOrConditional <- <(e1 / conditionalExpr)> */
		func() bool {
			position3, tokenIndex3, depth3 := position, tokenIndex, depth
			{
				position4 := position
				depth++
				{
					position5, tokenIndex5, depth5 := position, tokenIndex, depth
					if !_rules[rulee1]() {
						goto l6
					}
					goto l5
				l6:
					position, tokenIndex, depth = position5, tokenIndex5, depth5
					if !_rules[ruleconditionalExpr]() {
						goto l3
					}
				}
			l5:
				depth--
				add(ruleexprOrConditional, position4)
			}
			return true
		l3:
			position, tokenIndex, depth = position3, tokenIndex3, depth3
			return false
		},
		/* 2 conditionalExpr <- <(if b Action0 then exprOrConditional Action1 else exprOrConditional)+> */
		func() bool {
			position7, tokenIndex7, depth7 := position, tokenIndex, depth
			{
				position8 := position
				depth++
				if !_rules[ruleif]() {
					goto l7
				}
				if !_rules[ruleb]() {
					goto l7
				}
				if !_rules[ruleAction0]() {
					goto l7
				}
				if !_rules[rulethen]() {
					goto l7
				}
				if !_rules[ruleexprOrConditional]() {
					goto l7
				}
				if !_rules[ruleAction1]() {
					goto l7
				}
				if !_rules[ruleelse]() {
					goto l7
				}
				if !_rules[ruleexprOrConditional]() {
					goto l7
				}
			l9:
				{
					position10, tokenIndex10, depth10 := position, tokenIndex, depth
					if !_rules[ruleif]() {
						goto l10
					}
					if !_rules[ruleb]() {
						goto l10
					}
					if !_rules[ruleAction0]() {
						goto l10
					}
					if !_rules[rulethen]() {
						goto l10
					}
					if !_rules[ruleexprOrConditional]() {
						goto l10
					}
					if !_rules[ruleAction1]() {
						goto l10
					}
					if !_rules[ruleelse]() {
						goto l10
					}
					if !_rules[ruleexprOrConditional]() {
						goto l10
					}
					goto l9
				l10:
					position, tokenIndex, depth = position10, tokenIndex10, depth10
				}
				depth--
				add(ruleconditionalExpr, position8)
			}
			return true
		l7:
			position, tokenIndex, depth = position7, tokenIndex7, depth7
			return false
		},
		/* 3 b <- <(b1 ((and b1 Action2) / (or b1 Action3))*)> */
		func() bool {
			position11, tokenIndex11, depth11 := position, tokenIndex, depth
			{
				position12 := position
				depth++
				if !_rules[ruleb1]() {
					goto l11
				}
			l13:
				{
					position14, tokenIndex14, depth14 := position, tokenIndex, depth
					{
						position15, tokenIndex15, depth15 := position, tokenIndex, depth
						if !_rules[ruleand]() {
							goto l16
						}
						if !_rules[ruleb1]() {
							goto l16
						}
						if !_rules[ruleAction2]() {
							goto l16
						}
						goto l15
					l16:
						position, tokenIndex, depth = position15, tokenIndex15, depth15
						if !_rules[ruleor]() {
							goto l14
						}
						if !_rules[ruleb1]() {
							goto l14
						}
						if !_rules[ruleAction3]() {
							goto l14
						}
					}
				l15:
					goto l13
				l14:
					position, tokenIndex, depth = position14, tokenIndex14, depth14
				}
				depth--
				add(ruleb, position12)
			}
			return true
		l11:
			position, tokenIndex, depth = position11, tokenIndex11, depth11
			return false
		},
		/* 4 b1 <- <((not b2 Action4) / b2 / numericalComparison / stringComparison / timeComparison)> */
		func() bool {
			position17, tokenIndex17, depth17 := position, tokenIndex, depth
			{
				position18 := position
				depth++
				{
					position19, tokenIndex19, depth19 := position, tokenIndex, depth
					if !_rules[rulenot]() {
						goto l20
					}
					if !_rules[ruleb2]() {
						goto l20
					}
					if !_rules[ruleAction4]() {
						goto l20
					}
					goto l19
				l20:
					position, tokenIndex, depth = position19, tokenIndex19, depth19
					if !_rules[ruleb2]() {
						goto l21
					}
					goto l19
				l21:
					position, tokenIndex, depth = position19, tokenIndex19, depth19
					if !_rules[rulenumericalComparison]() {
						goto l22
					}
					goto l19
				l22:
					position, tokenIndex, depth = position19, tokenIndex19, depth19
					if !_rules[rulestringComparison]() {
						goto l23
					}
					goto l19
				l23:
					position, tokenIndex, depth = position19, tokenIndex19, depth19
					if !_rules[ruletimeComparison]() {
						goto l17
					}
				}
			l19:
				depth--
				add(ruleb1, position18)
			}
			return true
		l17:
			position, tokenIndex, depth = position17, tokenIndex17, depth17
			return false
		},
		/* 5 b2 <- <logicalComparison> */
		func() bool {
			position24, tokenIndex24, depth24 := position, tokenIndex, depth
			{
				position25 := position
				depth++
				if !_rules[rulelogicalComparison]() {
					goto l24
				}
				depth--
				add(ruleb2, position25)
			}
			return true
		l24:
			position, tokenIndex, depth = position24, tokenIndex24, depth24
			return false
		},
		/* 6 timeComparison <- <(indexT equal indexBegin Action5)> */
		func() bool {
			position26, tokenIndex26, depth26 := position, tokenIndex, depth
			{
				position27 := position
				depth++
				if !_rules[ruleindexT]() {
					goto l26
				}
				if !_rules[ruleequal]() {
					goto l26
				}
				if !_rules[ruleindexBegin]() {
					goto l26
				}
				if !_rules[ruleAction5]() {
					goto l26
				}
				depth--
				add(ruletimeComparison, position27)
			}
			return true
		l26:
			position, tokenIndex, depth = position26, tokenIndex26, depth26
			return false
		},
		/* 7 numericalComparison <- <(e1 ((equal e1 Action6) / (notEqual e1 Action7) / (greaterThan e1 Action8) / (greaterThanEqual e1 Action9) / (lessThan e1 Action10) / (lessThanEqual e1 Action11)))> */
		func() bool {
			position28, tokenIndex28, depth28 := position, tokenIndex, depth
			{
				position29 := position
				depth++
				if !_rules[rulee1]() {
					goto l28
				}
				{
					position30, tokenIndex30, depth30 := position, tokenIndex, depth
					if !_rules[ruleequal]() {
						goto l31
					}
					if !_rules[rulee1]() {
						goto l31
					}
					if !_rules[ruleAction6]() {
						goto l31
					}
					goto l30
				l31:
					position, tokenIndex, depth = position30, tokenIndex30, depth30
					if !_rules[rulenotEqual]() {
						goto l32
					}
					if !_rules[rulee1]() {
						goto l32
					}
					if !_rules[ruleAction7]() {
						goto l32
					}
					goto l30
				l32:
					position, tokenIndex, depth = position30, tokenIndex30, depth30
					if !_rules[rulegreaterThan]() {
						goto l33
					}
					if !_rules[rulee1]() {
						goto l33
					}
					if !_rules[ruleAction8]() {
						goto l33
					}
					goto l30
				l33:
					position, tokenIndex, depth = position30, tokenIndex30, depth30
					if !_rules[rulegreaterThanEqual]() {
						goto l34
					}
					if !_rules[rulee1]() {
						goto l34
					}
					if !_rules[ruleAction9]() {
						goto l34
					}
					goto l30
				l34:
					position, tokenIndex, depth = position30, tokenIndex30, depth30
					if !_rules[rulelessThan]() {
						goto l35
					}
					if !_rules[rulee1]() {
						goto l35
					}
					if !_rules[ruleAction10]() {
						goto l35
					}
					goto l30
				l35:
					position, tokenIndex, depth = position30, tokenIndex30, depth30
					if !_rules[rulelessThanEqual]() {
						goto l28
					}
					if !_rules[rulee1]() {
						goto l28
					}
					if !_rules[ruleAction11]() {
						goto l28
					}
				}
			l30:
				depth--
				add(rulenumericalComparison, position29)
			}
			return true
		l28:
			position, tokenIndex, depth = position28, tokenIndex28, depth28
			return false
		},
		/* 8 logicalComparison <- <(booleanValue ((equal booleanValue Action12) / (notEqual booleanValue Action13)))> */
		func() bool {
			position36, tokenIndex36, depth36 := position, tokenIndex, depth
			{
				position37 := position
				depth++
				if !_rules[rulebooleanValue]() {
					goto l36
				}
				{
					position38, tokenIndex38, depth38 := position, tokenIndex, depth
					if !_rules[ruleequal]() {
						goto l39
					}
					if !_rules[rulebooleanValue]() {
						goto l39
					}
					if !_rules[ruleAction12]() {
						goto l39
					}
					goto l38
				l39:
					position, tokenIndex, depth = position38, tokenIndex38, depth38
					if !_rules[rulenotEqual]() {
						goto l36
					}
					if !_rules[rulebooleanValue]() {
						goto l36
					}
					if !_rules[ruleAction13]() {
						goto l36
					}
				}
			l38:
				depth--
				add(rulelogicalComparison, position37)
			}
			return true
		l36:
			position, tokenIndex, depth = position36, tokenIndex36, depth36
			return false
		},
		/* 9 stringComparison <- <(stringExpression ((equal stringExpression Action14) / (notEqual stringExpression Action15)))> */
		func() bool {
			position40, tokenIndex40, depth40 := position, tokenIndex, depth
			{
				position41 := position
				depth++
				if !_rules[rulestringExpression]() {
					goto l40
				}
				{
					position42, tokenIndex42, depth42 := position, tokenIndex, depth
					if !_rules[ruleequal]() {
						goto l43
					}
					if !_rules[rulestringExpression]() {
						goto l43
					}
					if !_rules[ruleAction14]() {
						goto l43
					}
					goto l42
				l43:
					position, tokenIndex, depth = position42, tokenIndex42, depth42
					if !_rules[rulenotEqual]() {
						goto l40
					}
					if !_rules[rulestringExpression]() {
						goto l40
					}
					if !_rules[ruleAction15]() {
						goto l40
					}
				}
			l42:
				depth--
				add(rulestringComparison, position41)
			}
			return true
		l40:
			position, tokenIndex, depth = position40, tokenIndex40, depth40
			return false
		},
		/* 10 stringExpression <- <(stringValue / categoryIdentifier)> */
		func() bool {
			position44, tokenIndex44, depth44 := position, tokenIndex, depth
			{
				position45 := position
				depth++
				{
					position46, tokenIndex46, depth46 := position, tokenIndex, depth
					if !_rules[rulestringValue]() {
						goto l47
					}
					goto l46
				l47:
					position, tokenIndex, depth = position46, tokenIndex46, depth46
					if !_rules[rulecategoryIdentifier]() {
						goto l44
					}
				}
			l46:
				depth--
				add(rulestringExpression, position45)
			}
			return true
		l44:
			position, tokenIndex, depth = position44, tokenIndex44, depth44
			return false
		},
		/* 11 booleanValue <- <(true / false / (open b close))> */
		func() bool {
			position48, tokenIndex48, depth48 := position, tokenIndex, depth
			{
				position49 := position
				depth++
				{
					position50, tokenIndex50, depth50 := position, tokenIndex, depth
					if !_rules[ruletrue]() {
						goto l51
					}
					goto l50
				l51:
					position, tokenIndex, depth = position50, tokenIndex50, depth50
					if !_rules[rulefalse]() {
						goto l52
					}
					goto l50
				l52:
					position, tokenIndex, depth = position50, tokenIndex50, depth50
					if !_rules[ruleopen]() {
						goto l48
					}
					if !_rules[ruleb]() {
						goto l48
					}
					if !_rules[ruleclose]() {
						goto l48
					}
				}
			l50:
				depth--
				add(rulebooleanValue, position49)
			}
			return true
		l48:
			position, tokenIndex, depth = position48, tokenIndex48, depth48
			return false
		},
		/* 12 e1 <- <(e2 ((add e2 Action16) / (minus e2 Action17))*)> */
		func() bool {
			position53, tokenIndex53, depth53 := position, tokenIndex, depth
			{
				position54 := position
				depth++
				if !_rules[rulee2]() {
					goto l53
				}
			l55:
				{
					position56, tokenIndex56, depth56 := position, tokenIndex, depth
					{
						position57, tokenIndex57, depth57 := position, tokenIndex, depth
						if !_rules[ruleadd]() {
							goto l58
						}
						if !_rules[rulee2]() {
							goto l58
						}
						if !_rules[ruleAction16]() {
							goto l58
						}
						goto l57
					l58:
						position, tokenIndex, depth = position57, tokenIndex57, depth57
						if !_rules[ruleminus]() {
							goto l56
						}
						if !_rules[rulee2]() {
							goto l56
						}
						if !_rules[ruleAction17]() {
							goto l56
						}
					}
				l57:
					goto l55
				l56:
					position, tokenIndex, depth = position56, tokenIndex56, depth56
				}
				depth--
				add(rulee1, position54)
			}
			return true
		l53:
			position, tokenIndex, depth = position53, tokenIndex53, depth53
			return false
		},
		/* 13 e2 <- <(e3 ((multiply e3 Action18) / (divide e3 Action19) / (modulus e3 Action20))*)> */
		func() bool {
			position59, tokenIndex59, depth59 := position, tokenIndex, depth
			{
				position60 := position
				depth++
				if !_rules[rulee3]() {
					goto l59
				}
			l61:
				{
					position62, tokenIndex62, depth62 := position, tokenIndex, depth
					{
						position63, tokenIndex63, depth63 := position, tokenIndex, depth
						if !_rules[rulemultiply]() {
							goto l64
						}
						if !_rules[rulee3]() {
							goto l64
						}
						if !_rules[ruleAction18]() {
							goto l64
						}
						goto l63
					l64:
						position, tokenIndex, depth = position63, tokenIndex63, depth63
						if !_rules[ruledivide]() {
							goto l65
						}
						if !_rules[rulee3]() {
							goto l65
						}
						if !_rules[ruleAction19]() {
							goto l65
						}
						goto l63
					l65:
						position, tokenIndex, depth = position63, tokenIndex63, depth63
						if !_rules[rulemodulus]() {
							goto l62
						}
						if !_rules[rulee3]() {
							goto l62
						}
						if !_rules[ruleAction20]() {
							goto l62
						}
					}
				l63:
					goto l61
				l62:
					position, tokenIndex, depth = position62, tokenIndex62, depth62
				}
				depth--
				add(rulee2, position60)
			}
			return true
		l59:
			position, tokenIndex, depth = position59, tokenIndex59, depth59
			return false
		},
		/* 14 e3 <- <(e4 (exponentiation e4 Action21)*)> */
		func() bool {
			position66, tokenIndex66, depth66 := position, tokenIndex, depth
			{
				position67 := position
				depth++
				if !_rules[rulee4]() {
					goto l66
				}
			l68:
				{
					position69, tokenIndex69, depth69 := position, tokenIndex, depth
					if !_rules[ruleexponentiation]() {
						goto l69
					}
					if !_rules[rulee4]() {
						goto l69
					}
					if !_rules[ruleAction21]() {
						goto l69
					}
					goto l68
				l69:
					position, tokenIndex, depth = position69, tokenIndex69, depth69
				}
				depth--
				add(rulee3, position67)
			}
			return true
		l66:
			position, tokenIndex, depth = position66, tokenIndex66, depth66
			return false
		},
		/* 15 e4 <- <((minus value Action22) / value)> */
		func() bool {
			position70, tokenIndex70, depth70 := position, tokenIndex, depth
			{
				position71 := position
				depth++
				{
					position72, tokenIndex72, depth72 := position, tokenIndex, depth
					if !_rules[ruleminus]() {
						goto l73
					}
					if !_rules[rulevalue]() {
						goto l73
					}
					if !_rules[ruleAction22]() {
						goto l73
					}
					goto l72
				l73:
					position, tokenIndex, depth = position72, tokenIndex72, depth72
					if !_rules[rulevalue]() {
						goto l70
					}
				}
			l72:
				depth--
				add(rulee4, position71)
			}
			return true
		l70:
			position, tokenIndex, depth = position70, tokenIndex70, depth70
			return false
		},
		/* 16 value <- <((<([0-9] / '.')+> sp Action23) / identifier / (open e1 close))> */
		func() bool {
			position74, tokenIndex74, depth74 := position, tokenIndex, depth
			{
				position75 := position
				depth++
				{
					position76, tokenIndex76, depth76 := position, tokenIndex, depth
					{
						position78 := position
						depth++
						{
							position81, tokenIndex81, depth81 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l82
							}
							position++
							goto l81
						l82:
							position, tokenIndex, depth = position81, tokenIndex81, depth81
							if buffer[position] != rune('.') {
								goto l77
							}
							position++
						}
					l81:
					l79:
						{
							position80, tokenIndex80, depth80 := position, tokenIndex, depth
							{
								position83, tokenIndex83, depth83 := position, tokenIndex, depth
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l84
								}
								position++
								goto l83
							l84:
								position, tokenIndex, depth = position83, tokenIndex83, depth83
								if buffer[position] != rune('.') {
									goto l80
								}
								position++
							}
						l83:
							goto l79
						l80:
							position, tokenIndex, depth = position80, tokenIndex80, depth80
						}
						depth--
						add(rulePegText, position78)
					}
					if !_rules[rulesp]() {
						goto l77
					}
					if !_rules[ruleAction23]() {
						goto l77
					}
					goto l76
				l77:
					position, tokenIndex, depth = position76, tokenIndex76, depth76
					if !_rules[ruleidentifier]() {
						goto l85
					}
					goto l76
				l85:
					position, tokenIndex, depth = position76, tokenIndex76, depth76
					if !_rules[ruleopen]() {
						goto l74
					}
					if !_rules[rulee1]() {
						goto l74
					}
					if !_rules[ruleclose]() {
						goto l74
					}
				}
			l76:
				depth--
				add(rulevalue, position75)
			}
			return true
		l74:
			position, tokenIndex, depth = position74, tokenIndex74, depth74
			return false
		},
		/* 17 identifier <- <(specificIdentifier / generalIdentifier / thisIdentifier / functionCall)> */
		func() bool {
			position86, tokenIndex86, depth86 := position, tokenIndex, depth
			{
				position87 := position
				depth++
				{
					position88, tokenIndex88, depth88 := position, tokenIndex, depth
					if !_rules[rulespecificIdentifier]() {
						goto l89
					}
					goto l88
				l89:
					position, tokenIndex, depth = position88, tokenIndex88, depth88
					if !_rules[rulegeneralIdentifier]() {
						goto l90
					}
					goto l88
				l90:
					position, tokenIndex, depth = position88, tokenIndex88, depth88
					if !_rules[rulethisIdentifier]() {
						goto l91
					}
					goto l88
				l91:
					position, tokenIndex, depth = position88, tokenIndex88, depth88
					if !_rules[rulefunctionCall]() {
						goto l86
					}
				}
			l88:
				depth--
				add(ruleidentifier, position87)
			}
			return true
		l86:
			position, tokenIndex, depth = position86, tokenIndex86, depth86
			return false
		},
		/* 18 functionCall <- <(functionName open functionArgumentList close Action24)> */
		func() bool {
			position92, tokenIndex92, depth92 := position, tokenIndex, depth
			{
				position93 := position
				depth++
				if !_rules[rulefunctionName]() {
					goto l92
				}
				if !_rules[ruleopen]() {
					goto l92
				}
				if !_rules[rulefunctionArgumentList]() {
					goto l92
				}
				if !_rules[ruleclose]() {
					goto l92
				}
				if !_rules[ruleAction24]() {
					goto l92
				}
				depth--
				add(rulefunctionCall, position93)
			}
			return true
		l92:
			position, tokenIndex, depth = position92, tokenIndex92, depth92
			return false
		},
		/* 19 functionArgumentList <- <(functionArgument (comma functionArgument)*)> */
		func() bool {
			position94, tokenIndex94, depth94 := position, tokenIndex, depth
			{
				position95 := position
				depth++
				if !_rules[rulefunctionArgument]() {
					goto l94
				}
			l96:
				{
					position97, tokenIndex97, depth97 := position, tokenIndex, depth
					if !_rules[rulecomma]() {
						goto l97
					}
					if !_rules[rulefunctionArgument]() {
						goto l97
					}
					goto l96
				l97:
					position, tokenIndex, depth = position97, tokenIndex97, depth97
				}
				depth--
				add(rulefunctionArgumentList, position95)
			}
			return true
		l94:
			position, tokenIndex, depth = position94, tokenIndex94, depth94
			return false
		},
		/* 20 functionArgument <- <(wholeSeries Action25)> */
		func() bool {
			position98, tokenIndex98, depth98 := position, tokenIndex, depth
			{
				position99 := position
				depth++
				if !_rules[rulewholeSeries]() {
					goto l98
				}
				if !_rules[ruleAction25]() {
					goto l98
				}
				depth--
				add(rulefunctionArgument, position99)
			}
			return true
		l98:
			position, tokenIndex, depth = position98, tokenIndex98, depth98
			return false
		},
		/* 21 functionName <- <(<(([a-z] / [A-Z])+ ([a-z] / [A-Z] / [0-9])*)> Action26)> */
		func() bool {
			position100, tokenIndex100, depth100 := position, tokenIndex, depth
			{
				position101 := position
				depth++
				{
					position102 := position
					depth++
					{
						position105, tokenIndex105, depth105 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l106
						}
						position++
						goto l105
					l106:
						position, tokenIndex, depth = position105, tokenIndex105, depth105
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l100
						}
						position++
					}
				l105:
				l103:
					{
						position104, tokenIndex104, depth104 := position, tokenIndex, depth
						{
							position107, tokenIndex107, depth107 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l108
							}
							position++
							goto l107
						l108:
							position, tokenIndex, depth = position107, tokenIndex107, depth107
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l104
							}
							position++
						}
					l107:
						goto l103
					l104:
						position, tokenIndex, depth = position104, tokenIndex104, depth104
					}
				l109:
					{
						position110, tokenIndex110, depth110 := position, tokenIndex, depth
						{
							position111, tokenIndex111, depth111 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l112
							}
							position++
							goto l111
						l112:
							position, tokenIndex, depth = position111, tokenIndex111, depth111
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l113
							}
							position++
							goto l111
						l113:
							position, tokenIndex, depth = position111, tokenIndex111, depth111
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l110
							}
							position++
						}
					l111:
						goto l109
					l110:
						position, tokenIndex, depth = position110, tokenIndex110, depth110
					}
					depth--
					add(rulePegText, position102)
				}
				if !_rules[ruleAction26]() {
					goto l100
				}
				depth--
				add(rulefunctionName, position101)
			}
			return true
		l100:
			position, tokenIndex, depth = position100, tokenIndex100, depth100
			return false
		},
		/* 22 wholeSeries <- <(specificIdentifierRange / generalIdentifierRange / thisIdentifierRange)> */
		func() bool {
			position114, tokenIndex114, depth114 := position, tokenIndex, depth
			{
				position115 := position
				depth++
				{
					position116, tokenIndex116, depth116 := position, tokenIndex, depth
					if !_rules[rulespecificIdentifierRange]() {
						goto l117
					}
					goto l116
				l117:
					position, tokenIndex, depth = position116, tokenIndex116, depth116
					if !_rules[rulegeneralIdentifierRange]() {
						goto l118
					}
					goto l116
				l118:
					position, tokenIndex, depth = position116, tokenIndex116, depth116
					if !_rules[rulethisIdentifierRange]() {
						goto l114
					}
				}
			l116:
				depth--
				add(rulewholeSeries, position115)
			}
			return true
		l114:
			position, tokenIndex, depth = position114, tokenIndex114, depth114
			return false
		},
		/* 23 specificIdentifier <- <('v' 'a' 'l' <[0-9]+> Action27 timeIndex? sp)> */
		func() bool {
			position119, tokenIndex119, depth119 := position, tokenIndex, depth
			{
				position120 := position
				depth++
				if buffer[position] != rune('v') {
					goto l119
				}
				position++
				if buffer[position] != rune('a') {
					goto l119
				}
				position++
				if buffer[position] != rune('l') {
					goto l119
				}
				position++
				{
					position121 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l119
					}
					position++
				l122:
					{
						position123, tokenIndex123, depth123 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l123
						}
						position++
						goto l122
					l123:
						position, tokenIndex, depth = position123, tokenIndex123, depth123
					}
					depth--
					add(rulePegText, position121)
				}
				if !_rules[ruleAction27]() {
					goto l119
				}
				{
					position124, tokenIndex124, depth124 := position, tokenIndex, depth
					if !_rules[ruletimeIndex]() {
						goto l124
					}
					goto l125
				l124:
					position, tokenIndex, depth = position124, tokenIndex124, depth124
				}
			l125:
				if !_rules[rulesp]() {
					goto l119
				}
				depth--
				add(rulespecificIdentifier, position120)
			}
			return true
		l119:
			position, tokenIndex, depth = position119, tokenIndex119, depth119
			return false
		},
		/* 24 generalIdentifier <- <('v' 'a' 'l' Action28 timeIndex? sp)> */
		func() bool {
			position126, tokenIndex126, depth126 := position, tokenIndex, depth
			{
				position127 := position
				depth++
				if buffer[position] != rune('v') {
					goto l126
				}
				position++
				if buffer[position] != rune('a') {
					goto l126
				}
				position++
				if buffer[position] != rune('l') {
					goto l126
				}
				position++
				if !_rules[ruleAction28]() {
					goto l126
				}
				{
					position128, tokenIndex128, depth128 := position, tokenIndex, depth
					if !_rules[ruletimeIndex]() {
						goto l128
					}
					goto l129
				l128:
					position, tokenIndex, depth = position128, tokenIndex128, depth128
				}
			l129:
				if !_rules[rulesp]() {
					goto l126
				}
				depth--
				add(rulegeneralIdentifier, position127)
			}
			return true
		l126:
			position, tokenIndex, depth = position126, tokenIndex126, depth126
			return false
		},
		/* 25 thisIdentifier <- <('t' 'h' 'i' 's' Action29 timeIndex sp)> */
		func() bool {
			position130, tokenIndex130, depth130 := position, tokenIndex, depth
			{
				position131 := position
				depth++
				if buffer[position] != rune('t') {
					goto l130
				}
				position++
				if buffer[position] != rune('h') {
					goto l130
				}
				position++
				if buffer[position] != rune('i') {
					goto l130
				}
				position++
				if buffer[position] != rune('s') {
					goto l130
				}
				position++
				if !_rules[ruleAction29]() {
					goto l130
				}
				if !_rules[ruletimeIndex]() {
					goto l130
				}
				if !_rules[rulesp]() {
					goto l130
				}
				depth--
				add(rulethisIdentifier, position131)
			}
			return true
		l130:
			position, tokenIndex, depth = position130, tokenIndex130, depth130
			return false
		},
		/* 26 specificIdentifierRange <- <('v' 'a' 'l' <[0-9]+> Action30 timeRange sp)> */
		func() bool {
			position132, tokenIndex132, depth132 := position, tokenIndex, depth
			{
				position133 := position
				depth++
				if buffer[position] != rune('v') {
					goto l132
				}
				position++
				if buffer[position] != rune('a') {
					goto l132
				}
				position++
				if buffer[position] != rune('l') {
					goto l132
				}
				position++
				{
					position134 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l132
					}
					position++
				l135:
					{
						position136, tokenIndex136, depth136 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l136
						}
						position++
						goto l135
					l136:
						position, tokenIndex, depth = position136, tokenIndex136, depth136
					}
					depth--
					add(rulePegText, position134)
				}
				if !_rules[ruleAction30]() {
					goto l132
				}
				if !_rules[ruletimeRange]() {
					goto l132
				}
				if !_rules[rulesp]() {
					goto l132
				}
				depth--
				add(rulespecificIdentifierRange, position133)
			}
			return true
		l132:
			position, tokenIndex, depth = position132, tokenIndex132, depth132
			return false
		},
		/* 27 generalIdentifierRange <- <('v' 'a' 'l' Action31 timeRange sp)> */
		func() bool {
			position137, tokenIndex137, depth137 := position, tokenIndex, depth
			{
				position138 := position
				depth++
				if buffer[position] != rune('v') {
					goto l137
				}
				position++
				if buffer[position] != rune('a') {
					goto l137
				}
				position++
				if buffer[position] != rune('l') {
					goto l137
				}
				position++
				if !_rules[ruleAction31]() {
					goto l137
				}
				if !_rules[ruletimeRange]() {
					goto l137
				}
				if !_rules[rulesp]() {
					goto l137
				}
				depth--
				add(rulegeneralIdentifierRange, position138)
			}
			return true
		l137:
			position, tokenIndex, depth = position137, tokenIndex137, depth137
			return false
		},
		/* 28 thisIdentifierRange <- <('t' 'h' 'i' 's' Action32 timeRange sp)> */
		func() bool {
			position139, tokenIndex139, depth139 := position, tokenIndex, depth
			{
				position140 := position
				depth++
				if buffer[position] != rune('t') {
					goto l139
				}
				position++
				if buffer[position] != rune('h') {
					goto l139
				}
				position++
				if buffer[position] != rune('i') {
					goto l139
				}
				position++
				if buffer[position] != rune('s') {
					goto l139
				}
				position++
				if !_rules[ruleAction32]() {
					goto l139
				}
				if !_rules[ruletimeRange]() {
					goto l139
				}
				if !_rules[rulesp]() {
					goto l139
				}
				depth--
				add(rulethisIdentifierRange, position140)
			}
			return true
		l139:
			position, tokenIndex, depth = position139, tokenIndex139, depth139
			return false
		},
		/* 29 categoryIdentifier <- <('c' 'a' 't' 'e' 'g' 'o' 'r' 'y' Action33 sp)> */
		func() bool {
			position141, tokenIndex141, depth141 := position, tokenIndex, depth
			{
				position142 := position
				depth++
				if buffer[position] != rune('c') {
					goto l141
				}
				position++
				if buffer[position] != rune('a') {
					goto l141
				}
				position++
				if buffer[position] != rune('t') {
					goto l141
				}
				position++
				if buffer[position] != rune('e') {
					goto l141
				}
				position++
				if buffer[position] != rune('g') {
					goto l141
				}
				position++
				if buffer[position] != rune('o') {
					goto l141
				}
				position++
				if buffer[position] != rune('r') {
					goto l141
				}
				position++
				if buffer[position] != rune('y') {
					goto l141
				}
				position++
				if !_rules[ruleAction33]() {
					goto l141
				}
				if !_rules[rulesp]() {
					goto l141
				}
				depth--
				add(rulecategoryIdentifier, position142)
			}
			return true
		l141:
			position, tokenIndex, depth = position141, tokenIndex141, depth141
			return false
		},
		/* 30 stringValue <- <(quote <(!('"' / '\\' / '\n' / '\r') .)*> quote Action34 sp)> */
		func() bool {
			position143, tokenIndex143, depth143 := position, tokenIndex, depth
			{
				position144 := position
				depth++
				if !_rules[rulequote]() {
					goto l143
				}
				{
					position145 := position
					depth++
				l146:
					{
						position147, tokenIndex147, depth147 := position, tokenIndex, depth
						{
							position148, tokenIndex148, depth148 := position, tokenIndex, depth
							{
								position149, tokenIndex149, depth149 := position, tokenIndex, depth
								if buffer[position] != rune('"') {
									goto l150
								}
								position++
								goto l149
							l150:
								position, tokenIndex, depth = position149, tokenIndex149, depth149
								if buffer[position] != rune('\\') {
									goto l151
								}
								position++
								goto l149
							l151:
								position, tokenIndex, depth = position149, tokenIndex149, depth149
								if buffer[position] != rune('\n') {
									goto l152
								}
								position++
								goto l149
							l152:
								position, tokenIndex, depth = position149, tokenIndex149, depth149
								if buffer[position] != rune('\r') {
									goto l148
								}
								position++
							}
						l149:
							goto l147
						l148:
							position, tokenIndex, depth = position148, tokenIndex148, depth148
						}
						if !matchDot() {
							goto l147
						}
						goto l146
					l147:
						position, tokenIndex, depth = position147, tokenIndex147, depth147
					}
					depth--
					add(rulePegText, position145)
				}
				if !_rules[rulequote]() {
					goto l143
				}
				if !_rules[ruleAction34]() {
					goto l143
				}
				if !_rules[rulesp]() {
					goto l143
				}
				depth--
				add(rulestringValue, position144)
			}
			return true
		l143:
			position, tokenIndex, depth = position143, tokenIndex143, depth143
			return false
		},
		/* 31 timeRange <- <(openIndex indexComputation colon indexComputation closeIndex Action35)> */
		func() bool {
			position153, tokenIndex153, depth153 := position, tokenIndex, depth
			{
				position154 := position
				depth++
				if !_rules[ruleopenIndex]() {
					goto l153
				}
				if !_rules[ruleindexComputation]() {
					goto l153
				}
				if !_rules[rulecolon]() {
					goto l153
				}
				if !_rules[ruleindexComputation]() {
					goto l153
				}
				if !_rules[rulecloseIndex]() {
					goto l153
				}
				if !_rules[ruleAction35]() {
					goto l153
				}
				depth--
				add(ruletimeRange, position154)
			}
			return true
		l153:
			position, tokenIndex, depth = position153, tokenIndex153, depth153
			return false
		},
		/* 32 timeIndex <- <(openIndex indexComputation closeIndex)> */
		func() bool {
			position155, tokenIndex155, depth155 := position, tokenIndex, depth
			{
				position156 := position
				depth++
				if !_rules[ruleopenIndex]() {
					goto l155
				}
				if !_rules[ruleindexComputation]() {
					goto l155
				}
				if !_rules[rulecloseIndex]() {
					goto l155
				}
				depth--
				add(ruletimeIndex, position156)
			}
			return true
		l155:
			position, tokenIndex, depth = position155, tokenIndex155, depth155
			return false
		},
		/* 33 indexComputation <- <(indexExpr ((add indexExpr Action36) / (minus indexExpr Action37))*)> */
		func() bool {
			position157, tokenIndex157, depth157 := position, tokenIndex, depth
			{
				position158 := position
				depth++
				if !_rules[ruleindexExpr]() {
					goto l157
				}
			l159:
				{
					position160, tokenIndex160, depth160 := position, tokenIndex, depth
					{
						position161, tokenIndex161, depth161 := position, tokenIndex, depth
						if !_rules[ruleadd]() {
							goto l162
						}
						if !_rules[ruleindexExpr]() {
							goto l162
						}
						if !_rules[ruleAction36]() {
							goto l162
						}
						goto l161
					l162:
						position, tokenIndex, depth = position161, tokenIndex161, depth161
						if !_rules[ruleminus]() {
							goto l160
						}
						if !_rules[ruleindexExpr]() {
							goto l160
						}
						if !_rules[ruleAction37]() {
							goto l160
						}
					}
				l161:
					goto l159
				l160:
					position, tokenIndex, depth = position160, tokenIndex160, depth160
				}
				depth--
				add(ruleindexComputation, position158)
			}
			return true
		l157:
			position, tokenIndex, depth = position157, tokenIndex157, depth157
			return false
		},
		/* 34 indexExpr <- <((indexBegin Action38) / (indexEnd Action39) / (indexT Action40) / (<[0-9]+> Action41))> */
		func() bool {
			position163, tokenIndex163, depth163 := position, tokenIndex, depth
			{
				position164 := position
				depth++
				{
					position165, tokenIndex165, depth165 := position, tokenIndex, depth
					if !_rules[ruleindexBegin]() {
						goto l166
					}
					if !_rules[ruleAction38]() {
						goto l166
					}
					goto l165
				l166:
					position, tokenIndex, depth = position165, tokenIndex165, depth165
					if !_rules[ruleindexEnd]() {
						goto l167
					}
					if !_rules[ruleAction39]() {
						goto l167
					}
					goto l165
				l167:
					position, tokenIndex, depth = position165, tokenIndex165, depth165
					if !_rules[ruleindexT]() {
						goto l168
					}
					if !_rules[ruleAction40]() {
						goto l168
					}
					goto l165
				l168:
					position, tokenIndex, depth = position165, tokenIndex165, depth165
					{
						position169 := position
						depth++
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l163
						}
						position++
					l170:
						{
							position171, tokenIndex171, depth171 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l171
							}
							position++
							goto l170
						l171:
							position, tokenIndex, depth = position171, tokenIndex171, depth171
						}
						depth--
						add(rulePegText, position169)
					}
					if !_rules[ruleAction41]() {
						goto l163
					}
				}
			l165:
				depth--
				add(ruleindexExpr, position164)
			}
			return true
		l163:
			position, tokenIndex, depth = position163, tokenIndex163, depth163
			return false
		},
		/* 35 indexBegin <- <('b' 'e' 'g' 'i' 'n' sp)> */
		func() bool {
			position172, tokenIndex172, depth172 := position, tokenIndex, depth
			{
				position173 := position
				depth++
				if buffer[position] != rune('b') {
					goto l172
				}
				position++
				if buffer[position] != rune('e') {
					goto l172
				}
				position++
				if buffer[position] != rune('g') {
					goto l172
				}
				position++
				if buffer[position] != rune('i') {
					goto l172
				}
				position++
				if buffer[position] != rune('n') {
					goto l172
				}
				position++
				if !_rules[rulesp]() {
					goto l172
				}
				depth--
				add(ruleindexBegin, position173)
			}
			return true
		l172:
			position, tokenIndex, depth = position172, tokenIndex172, depth172
			return false
		},
		/* 36 indexEnd <- <('e' 'n' 'd' sp)> */
		func() bool {
			position174, tokenIndex174, depth174 := position, tokenIndex, depth
			{
				position175 := position
				depth++
				if buffer[position] != rune('e') {
					goto l174
				}
				position++
				if buffer[position] != rune('n') {
					goto l174
				}
				position++
				if buffer[position] != rune('d') {
					goto l174
				}
				position++
				if !_rules[rulesp]() {
					goto l174
				}
				depth--
				add(ruleindexEnd, position175)
			}
			return true
		l174:
			position, tokenIndex, depth = position174, tokenIndex174, depth174
			return false
		},
		/* 37 indexT <- <('t' sp)> */
		func() bool {
			position176, tokenIndex176, depth176 := position, tokenIndex, depth
			{
				position177 := position
				depth++
				if buffer[position] != rune('t') {
					goto l176
				}
				position++
				if !_rules[rulesp]() {
					goto l176
				}
				depth--
				add(ruleindexT, position177)
			}
			return true
		l176:
			position, tokenIndex, depth = position176, tokenIndex176, depth176
			return false
		},
		/* 38 openIndex <- <('[' sp)> */
		func() bool {
			position178, tokenIndex178, depth178 := position, tokenIndex, depth
			{
				position179 := position
				depth++
				if buffer[position] != rune('[') {
					goto l178
				}
				position++
				if !_rules[rulesp]() {
					goto l178
				}
				depth--
				add(ruleopenIndex, position179)
			}
			return true
		l178:
			position, tokenIndex, depth = position178, tokenIndex178, depth178
			return false
		},
		/* 39 closeIndex <- <(']' sp)> */
		func() bool {
			position180, tokenIndex180, depth180 := position, tokenIndex, depth
			{
				position181 := position
				depth++
				if buffer[position] != rune(']') {
					goto l180
				}
				position++
				if !_rules[rulesp]() {
					goto l180
				}
				depth--
				add(rulecloseIndex, position181)
			}
			return true
		l180:
			position, tokenIndex, depth = position180, tokenIndex180, depth180
			return false
		},
		/* 40 if <- <('i' 'f' sp)> */
		func() bool {
			position182, tokenIndex182, depth182 := position, tokenIndex, depth
			{
				position183 := position
				depth++
				if buffer[position] != rune('i') {
					goto l182
				}
				position++
				if buffer[position] != rune('f') {
					goto l182
				}
				position++
				if !_rules[rulesp]() {
					goto l182
				}
				depth--
				add(ruleif, position183)
			}
			return true
		l182:
			position, tokenIndex, depth = position182, tokenIndex182, depth182
			return false
		},
		/* 41 then <- <('t' 'h' 'e' 'n' sp)> */
		func() bool {
			position184, tokenIndex184, depth184 := position, tokenIndex, depth
			{
				position185 := position
				depth++
				if buffer[position] != rune('t') {
					goto l184
				}
				position++
				if buffer[position] != rune('h') {
					goto l184
				}
				position++
				if buffer[position] != rune('e') {
					goto l184
				}
				position++
				if buffer[position] != rune('n') {
					goto l184
				}
				position++
				if !_rules[rulesp]() {
					goto l184
				}
				depth--
				add(rulethen, position185)
			}
			return true
		l184:
			position, tokenIndex, depth = position184, tokenIndex184, depth184
			return false
		},
		/* 42 else <- <('e' 'l' 's' 'e' sp)> */
		func() bool {
			position186, tokenIndex186, depth186 := position, tokenIndex, depth
			{
				position187 := position
				depth++
				if buffer[position] != rune('e') {
					goto l186
				}
				position++
				if buffer[position] != rune('l') {
					goto l186
				}
				position++
				if buffer[position] != rune('s') {
					goto l186
				}
				position++
				if buffer[position] != rune('e') {
					goto l186
				}
				position++
				if !_rules[rulesp]() {
					goto l186
				}
				depth--
				add(ruleelse, position187)
			}
			return true
		l186:
			position, tokenIndex, depth = position186, tokenIndex186, depth186
			return false
		},
		/* 43 true <- <('t' 'r' 'u' 'e' sp Action42)> */
		func() bool {
			position188, tokenIndex188, depth188 := position, tokenIndex, depth
			{
				position189 := position
				depth++
				if buffer[position] != rune('t') {
					goto l188
				}
				position++
				if buffer[position] != rune('r') {
					goto l188
				}
				position++
				if buffer[position] != rune('u') {
					goto l188
				}
				position++
				if buffer[position] != rune('e') {
					goto l188
				}
				position++
				if !_rules[rulesp]() {
					goto l188
				}
				if !_rules[ruleAction42]() {
					goto l188
				}
				depth--
				add(ruletrue, position189)
			}
			return true
		l188:
			position, tokenIndex, depth = position188, tokenIndex188, depth188
			return false
		},
		/* 44 false <- <('f' 'a' 'l' 's' 'e' sp Action43)> */
		func() bool {
			position190, tokenIndex190, depth190 := position, tokenIndex, depth
			{
				position191 := position
				depth++
				if buffer[position] != rune('f') {
					goto l190
				}
				position++
				if buffer[position] != rune('a') {
					goto l190
				}
				position++
				if buffer[position] != rune('l') {
					goto l190
				}
				position++
				if buffer[position] != rune('s') {
					goto l190
				}
				position++
				if buffer[position] != rune('e') {
					goto l190
				}
				position++
				if !_rules[rulesp]() {
					goto l190
				}
				if !_rules[ruleAction43]() {
					goto l190
				}
				depth--
				add(rulefalse, position191)
			}
			return true
		l190:
			position, tokenIndex, depth = position190, tokenIndex190, depth190
			return false
		},
		/* 45 equal <- <((('=' '=') / '=') sp)> */
		func() bool {
			position192, tokenIndex192, depth192 := position, tokenIndex, depth
			{
				position193 := position
				depth++
				{
					position194, tokenIndex194, depth194 := position, tokenIndex, depth
					if buffer[position] != rune('=') {
						goto l195
					}
					position++
					if buffer[position] != rune('=') {
						goto l195
					}
					position++
					goto l194
				l195:
					position, tokenIndex, depth = position194, tokenIndex194, depth194
					if buffer[position] != rune('=') {
						goto l192
					}
					position++
				}
			l194:
				if !_rules[rulesp]() {
					goto l192
				}
				depth--
				add(ruleequal, position193)
			}
			return true
		l192:
			position, tokenIndex, depth = position192, tokenIndex192, depth192
			return false
		},
		/* 46 notEqual <- <((('!' '=') / ('<' '>')) sp)> */
		func() bool {
			position196, tokenIndex196, depth196 := position, tokenIndex, depth
			{
				position197 := position
				depth++
				{
					position198, tokenIndex198, depth198 := position, tokenIndex, depth
					if buffer[position] != rune('!') {
						goto l199
					}
					position++
					if buffer[position] != rune('=') {
						goto l199
					}
					position++
					goto l198
				l199:
					position, tokenIndex, depth = position198, tokenIndex198, depth198
					if buffer[position] != rune('<') {
						goto l196
					}
					position++
					if buffer[position] != rune('>') {
						goto l196
					}
					position++
				}
			l198:
				if !_rules[rulesp]() {
					goto l196
				}
				depth--
				add(rulenotEqual, position197)
			}
			return true
		l196:
			position, tokenIndex, depth = position196, tokenIndex196, depth196
			return false
		},
		/* 47 greaterThan <- <('>' sp)> */
		func() bool {
			position200, tokenIndex200, depth200 := position, tokenIndex, depth
			{
				position201 := position
				depth++
				if buffer[position] != rune('>') {
					goto l200
				}
				position++
				if !_rules[rulesp]() {
					goto l200
				}
				depth--
				add(rulegreaterThan, position201)
			}
			return true
		l200:
			position, tokenIndex, depth = position200, tokenIndex200, depth200
			return false
		},
		/* 48 greaterThanEqual <- <('>' '=' sp)> */
		func() bool {
			position202, tokenIndex202, depth202 := position, tokenIndex, depth
			{
				position203 := position
				depth++
				if buffer[position] != rune('>') {
					goto l202
				}
				position++
				if buffer[position] != rune('=') {
					goto l202
				}
				position++
				if !_rules[rulesp]() {
					goto l202
				}
				depth--
				add(rulegreaterThanEqual, position203)
			}
			return true
		l202:
			position, tokenIndex, depth = position202, tokenIndex202, depth202
			return false
		},
		/* 49 lessThan <- <('<' sp)> */
		func() bool {
			position204, tokenIndex204, depth204 := position, tokenIndex, depth
			{
				position205 := position
				depth++
				if buffer[position] != rune('<') {
					goto l204
				}
				position++
				if !_rules[rulesp]() {
					goto l204
				}
				depth--
				add(rulelessThan, position205)
			}
			return true
		l204:
			position, tokenIndex, depth = position204, tokenIndex204, depth204
			return false
		},
		/* 50 lessThanEqual <- <('<' '=' sp)> */
		func() bool {
			position206, tokenIndex206, depth206 := position, tokenIndex, depth
			{
				position207 := position
				depth++
				if buffer[position] != rune('<') {
					goto l206
				}
				position++
				if buffer[position] != rune('=') {
					goto l206
				}
				position++
				if !_rules[rulesp]() {
					goto l206
				}
				depth--
				add(rulelessThanEqual, position207)
			}
			return true
		l206:
			position, tokenIndex, depth = position206, tokenIndex206, depth206
			return false
		},
		/* 51 not <- <((('n' 'o' 't') / '!') sp)> */
		func() bool {
			position208, tokenIndex208, depth208 := position, tokenIndex, depth
			{
				position209 := position
				depth++
				{
					position210, tokenIndex210, depth210 := position, tokenIndex, depth
					if buffer[position] != rune('n') {
						goto l211
					}
					position++
					if buffer[position] != rune('o') {
						goto l211
					}
					position++
					if buffer[position] != rune('t') {
						goto l211
					}
					position++
					goto l210
				l211:
					position, tokenIndex, depth = position210, tokenIndex210, depth210
					if buffer[position] != rune('!') {
						goto l208
					}
					position++
				}
			l210:
				if !_rules[rulesp]() {
					goto l208
				}
				depth--
				add(rulenot, position209)
			}
			return true
		l208:
			position, tokenIndex, depth = position208, tokenIndex208, depth208
			return false
		},
		/* 52 and <- <((('a' 'n' 'd') / ('&' '&')) sp)> */
		func() bool {
			position212, tokenIndex212, depth212 := position, tokenIndex, depth
			{
				position213 := position
				depth++
				{
					position214, tokenIndex214, depth214 := position, tokenIndex, depth
					if buffer[position] != rune('a') {
						goto l215
					}
					position++
					if buffer[position] != rune('n') {
						goto l215
					}
					position++
					if buffer[position] != rune('d') {
						goto l215
					}
					position++
					goto l214
				l215:
					position, tokenIndex, depth = position214, tokenIndex214, depth214
					if buffer[position] != rune('&') {
						goto l212
					}
					position++
					if buffer[position] != rune('&') {
						goto l212
					}
					position++
				}
			l214:
				if !_rules[rulesp]() {
					goto l212
				}
				depth--
				add(ruleand, position213)
			}
			return true
		l212:
			position, tokenIndex, depth = position212, tokenIndex212, depth212
			return false
		},
		/* 53 or <- <((('o' 'r') / ('|' '|')) sp)> */
		func() bool {
			position216, tokenIndex216, depth216 := position, tokenIndex, depth
			{
				position217 := position
				depth++
				{
					position218, tokenIndex218, depth218 := position, tokenIndex, depth
					if buffer[position] != rune('o') {
						goto l219
					}
					position++
					if buffer[position] != rune('r') {
						goto l219
					}
					position++
					goto l218
				l219:
					position, tokenIndex, depth = position218, tokenIndex218, depth218
					if buffer[position] != rune('|') {
						goto l216
					}
					position++
					if buffer[position] != rune('|') {
						goto l216
					}
					position++
				}
			l218:
				if !_rules[rulesp]() {
					goto l216
				}
				depth--
				add(ruleor, position217)
			}
			return true
		l216:
			position, tokenIndex, depth = position216, tokenIndex216, depth216
			return false
		},
		/* 54 add <- <('+' sp)> */
		func() bool {
			position220, tokenIndex220, depth220 := position, tokenIndex, depth
			{
				position221 := position
				depth++
				if buffer[position] != rune('+') {
					goto l220
				}
				position++
				if !_rules[rulesp]() {
					goto l220
				}
				depth--
				add(ruleadd, position221)
			}
			return true
		l220:
			position, tokenIndex, depth = position220, tokenIndex220, depth220
			return false
		},
		/* 55 minus <- <('-' sp)> */
		func() bool {
			position222, tokenIndex222, depth222 := position, tokenIndex, depth
			{
				position223 := position
				depth++
				if buffer[position] != rune('-') {
					goto l222
				}
				position++
				if !_rules[rulesp]() {
					goto l222
				}
				depth--
				add(ruleminus, position223)
			}
			return true
		l222:
			position, tokenIndex, depth = position222, tokenIndex222, depth222
			return false
		},
		/* 56 multiply <- <('*' sp)> */
		func() bool {
			position224, tokenIndex224, depth224 := position, tokenIndex, depth
			{
				position225 := position
				depth++
				if buffer[position] != rune('*') {
					goto l224
				}
				position++
				if !_rules[rulesp]() {
					goto l224
				}
				depth--
				add(rulemultiply, position225)
			}
			return true
		l224:
			position, tokenIndex, depth = position224, tokenIndex224, depth224
			return false
		},
		/* 57 divide <- <('/' sp)> */
		func() bool {
			position226, tokenIndex226, depth226 := position, tokenIndex, depth
			{
				position227 := position
				depth++
				if buffer[position] != rune('/') {
					goto l226
				}
				position++
				if !_rules[rulesp]() {
					goto l226
				}
				depth--
				add(ruledivide, position227)
			}
			return true
		l226:
			position, tokenIndex, depth = position226, tokenIndex226, depth226
			return false
		},
		/* 58 modulus <- <('%' sp)> */
		func() bool {
			position228, tokenIndex228, depth228 := position, tokenIndex, depth
			{
				position229 := position
				depth++
				if buffer[position] != rune('%') {
					goto l228
				}
				position++
				if !_rules[rulesp]() {
					goto l228
				}
				depth--
				add(rulemodulus, position229)
			}
			return true
		l228:
			position, tokenIndex, depth = position228, tokenIndex228, depth228
			return false
		},
		/* 59 exponentiation <- <('^' sp)> */
		func() bool {
			position230, tokenIndex230, depth230 := position, tokenIndex, depth
			{
				position231 := position
				depth++
				if buffer[position] != rune('^') {
					goto l230
				}
				position++
				if !_rules[rulesp]() {
					goto l230
				}
				depth--
				add(ruleexponentiation, position231)
			}
			return true
		l230:
			position, tokenIndex, depth = position230, tokenIndex230, depth230
			return false
		},
		/* 60 open <- <('(' sp)> */
		func() bool {
			position232, tokenIndex232, depth232 := position, tokenIndex, depth
			{
				position233 := position
				depth++
				if buffer[position] != rune('(') {
					goto l232
				}
				position++
				if !_rules[rulesp]() {
					goto l232
				}
				depth--
				add(ruleopen, position233)
			}
			return true
		l232:
			position, tokenIndex, depth = position232, tokenIndex232, depth232
			return false
		},
		/* 61 close <- <(')' sp)> */
		func() bool {
			position234, tokenIndex234, depth234 := position, tokenIndex, depth
			{
				position235 := position
				depth++
				if buffer[position] != rune(')') {
					goto l234
				}
				position++
				if !_rules[rulesp]() {
					goto l234
				}
				depth--
				add(ruleclose, position235)
			}
			return true
		l234:
			position, tokenIndex, depth = position234, tokenIndex234, depth234
			return false
		},
		/* 62 comma <- <(',' sp)> */
		func() bool {
			position236, tokenIndex236, depth236 := position, tokenIndex, depth
			{
				position237 := position
				depth++
				if buffer[position] != rune(',') {
					goto l236
				}
				position++
				if !_rules[rulesp]() {
					goto l236
				}
				depth--
				add(rulecomma, position237)
			}
			return true
		l236:
			position, tokenIndex, depth = position236, tokenIndex236, depth236
			return false
		},
		/* 63 quote <- <('"' sp)> */
		func() bool {
			position238, tokenIndex238, depth238 := position, tokenIndex, depth
			{
				position239 := position
				depth++
				if buffer[position] != rune('"') {
					goto l238
				}
				position++
				if !_rules[rulesp]() {
					goto l238
				}
				depth--
				add(rulequote, position239)
			}
			return true
		l238:
			position, tokenIndex, depth = position238, tokenIndex238, depth238
			return false
		},
		/* 64 colon <- <(':' sp)> */
		func() bool {
			position240, tokenIndex240, depth240 := position, tokenIndex, depth
			{
				position241 := position
				depth++
				if buffer[position] != rune(':') {
					goto l240
				}
				position++
				if !_rules[rulesp]() {
					goto l240
				}
				depth--
				add(rulecolon, position241)
			}
			return true
		l240:
			position, tokenIndex, depth = position240, tokenIndex240, depth240
			return false
		},
		/* 65 sp <- <(' ' / '\t')*> */
		func() bool {
			{
				position243 := position
				depth++
			l244:
				{
					position245, tokenIndex245, depth245 := position, tokenIndex, depth
					{
						position246, tokenIndex246, depth246 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l247
						}
						position++
						goto l246
					l247:
						position, tokenIndex, depth = position246, tokenIndex246, depth246
						if buffer[position] != rune('\t') {
							goto l245
						}
						position++
					}
				l246:
					goto l244
				l245:
					position, tokenIndex, depth = position245, tokenIndex245, depth245
				}
				depth--
				add(rulesp, position243)
			}
			return true
		},
		/* 67 Action0 <- <{ p.AddOperator(TypeThen) }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 68 Action1 <- <{ p.AddOperator(TypeElse) }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 69 Action2 <- <{ p.AddOperator(TypeAnd) }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 70 Action3 <- <{ p.AddOperator(TypeOr) }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 71 Action4 <- <{ p.AddOperator(TypeNot) }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 72 Action5 <- <{ p.AddOperator(TypeTimeEqual) }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
		/* 73 Action6 <- <{ p.AddOperator(TypeEqual) }> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
		/* 74 Action7 <- <{ p.AddOperator(TypeNotEqual) }> */
		func() bool {
			{
				add(ruleAction7, position)
			}
			return true
		},
		/* 75 Action8 <- <{ p.AddOperator(TypeGreaterThan) }> */
		func() bool {
			{
				add(ruleAction8, position)
			}
			return true
		},
		/* 76 Action9 <- <{ p.AddOperator(TypeGreaterThanEqual) }> */
		func() bool {
			{
				add(ruleAction9, position)
			}
			return true
		},
		/* 77 Action10 <- <{ p.AddOperator(TypeLessThan) }> */
		func() bool {
			{
				add(ruleAction10, position)
			}
			return true
		},
		/* 78 Action11 <- <{ p.AddOperator(TypeLessThanEqual) }> */
		func() bool {
			{
				add(ruleAction11, position)
			}
			return true
		},
		/* 79 Action12 <- <{ p.AddOperator(TypeLogicalEqual) }> */
		func() bool {
			{
				add(ruleAction12, position)
			}
			return true
		},
		/* 80 Action13 <- <{ p.AddOperator(TypeLogicalNotEqual) }> */
		func() bool {
			{
				add(ruleAction13, position)
			}
			return true
		},
		/* 81 Action14 <- <{ p.AddOperator(TypeStringEqual)}> */
		func() bool {
			{
				add(ruleAction14, position)
			}
			return true
		},
		/* 82 Action15 <- <{ p.AddOperator(TypeStringNotEqual) }> */
		func() bool {
			{
				add(ruleAction15, position)
			}
			return true
		},
		/* 83 Action16 <- <{ p.AddOperator(TypeAdd) }> */
		func() bool {
			{
				add(ruleAction16, position)
			}
			return true
		},
		/* 84 Action17 <- <{ p.AddOperator(TypeSubtract) }> */
		func() bool {
			{
				add(ruleAction17, position)
			}
			return true
		},
		/* 85 Action18 <- <{ p.AddOperator(TypeMultiply) }> */
		func() bool {
			{
				add(ruleAction18, position)
			}
			return true
		},
		/* 86 Action19 <- <{ p.AddOperator(TypeDivide) }> */
		func() bool {
			{
				add(ruleAction19, position)
			}
			return true
		},
		/* 87 Action20 <- <{ p.AddOperator(TypeModulus) }> */
		func() bool {
			{
				add(ruleAction20, position)
			}
			return true
		},
		/* 88 Action21 <- <{ p.AddOperator(TypeExponentiation) }> */
		func() bool {
			{
				add(ruleAction21, position)
			}
			return true
		},
		/* 89 Action22 <- <{ p.AddOperator(TypeNegation) }> */
		func() bool {
			{
				add(ruleAction22, position)
			}
			return true
		},
		nil,
		/* 91 Action23 <- <{ p.AddValue(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction23, position)
			}
			return true
		},
		/* 92 Action24 <- <{ p.AddFunctionCall() }> */
		func() bool {
			{
				add(ruleAction24, position)
			}
			return true
		},
		/* 93 Action25 <- <{ p.AddFunctionArgument() }> */
		func() bool {
			{
				add(ruleAction25, position)
			}
			return true
		},
		/* 94 Action26 <- <{ p.AddFunctionName(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction26, position)
			}
			return true
		},
		/* 95 Action27 <- <{ p.AddIdentifierSpecific(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction27, position)
			}
			return true
		},
		/* 96 Action28 <- <{ p.AddIdentifierGeneral() }> */
		func() bool {
			{
				add(ruleAction28, position)
			}
			return true
		},
		/* 97 Action29 <- <{ p.AddIdentifierThis() }> */
		func() bool {
			{
				add(ruleAction29, position)
			}
			return true
		},
		/* 98 Action30 <- <{ p.AddIdentifierSpecificRange(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction30, position)
			}
			return true
		},
		/* 99 Action31 <- <{ p.AddIdentifierGeneralRange() }> */
		func() bool {
			{
				add(ruleAction31, position)
			}
			return true
		},
		/* 100 Action32 <- <{ p.AddIdentifierThisRange() }> */
		func() bool {
			{
				add(ruleAction32, position)
			}
			return true
		},
		/* 101 Action33 <- <{ p.AddCategoryIdentifier() }> */
		func() bool {
			{
				add(ruleAction33, position)
			}
			return true
		},
		/* 102 Action34 <- <{ p.AddStringValue(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction34, position)
			}
			return true
		},
		/* 103 Action35 <- <{ p.AddIndexOperator(TypeTimeRange) }> */
		func() bool {
			{
				add(ruleAction35, position)
			}
			return true
		},
		/* 104 Action36 <- <{ p.AddIndexOperator(TypeAdd) }> */
		func() bool {
			{
				add(ruleAction36, position)
			}
			return true
		},
		/* 105 Action37 <- <{ p.AddIndexOperator(TypeSubtract) }> */
		func() bool {
			{
				add(ruleAction37, position)
			}
			return true
		},
		/* 106 Action38 <- <{ p.AddIndexOperator(TypeBegin) }> */
		func() bool {
			{
				add(ruleAction38, position)
			}
			return true
		},
		/* 107 Action39 <- <{ p.AddIndexOperator(TypeEnd) }> */
		func() bool {
			{
				add(ruleAction39, position)
			}
			return true
		},
		/* 108 Action40 <- <{ p.AddIndexOperator(TypeCurrentTime) }> */
		func() bool {
			{
				add(ruleAction40, position)
			}
			return true
		},
		/* 109 Action41 <- <{ p.AddIndexValue(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction41, position)
			}
			return true
		},
		/* 110 Action42 <- <{ p.AddOperator(TypeTrue) }> */
		func() bool {
			{
				add(ruleAction42, position)
			}
			return true
		},
		/* 111 Action43 <- <{ p.AddOperator(TypeFalse) }> */
		func() bool {
			{
				add(ruleAction43, position)
			}
			return true
		},
	}
	p.rules = _rules
}
