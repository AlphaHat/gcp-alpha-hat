package parse

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type Type uint8

const (
	TypeNumber Type = iota
	TypeString
	TypeNegation
	TypeAdd
	TypeSubtract
	TypeMultiply
	TypeDivide
	TypeModulus
	TypeExponentiation
	TypeIdentifierGeneral
	TypeIdentifierSpecific
	TypeIdentifierThis
	TypeIdentifierGeneralRange
	TypeIdentifierSpecificRange
	TypeIdentifierThisRange
	TypeTimeRange
	TypeIdentifierCategory
	TypeCurrentTime
	TypeBegin
	TypeEnd
	TypeThen
	TypeElse
	TypeLogicalEqual
	TypeLogicalNotEqual
	TypeTimeEqual
	TypeEqual
	TypeNotEqual
	TypeGreaterThan
	TypeGreaterThanEqual
	TypeLessThan
	TypeLessThanEqual
	TypeAnd
	TypeOr
	TypeTrue
	TypeFalse
	TypeNot
	TypeFunctionCall
	TypeStringEqual
	TypeStringNotEqual
)

type IndexCode struct {
	T   Type
	Int int
}

type ByteCode struct {
	T       Type
	Float   float64
	Int     int
	Str     string
	IndexOp []IndexCode
}

func (code *IndexCode) String() string {
	switch code.T {
	case TypeBegin:
		return "begin"
	case TypeEnd:
		return "end"
	case TypeCurrentTime:
		return "t"
	case TypeNumber:
		return fmt.Sprintf("%v", code.Int)
	case TypeAdd:
		return "+"
	case TypeNegation, TypeSubtract:
		return "-"
	case TypeTimeRange:
		return ":"
	}
	return "Unknown Type"
}

func (code *ByteCode) String() string {
	switch code.T {
	case TypeNumber:
		return fmt.Sprintf("%v", code.Float)
	case TypeAdd:
		return "+"
	case TypeNegation, TypeSubtract:
		return "-"
	case TypeMultiply:
		return "*"
	case TypeDivide:
		return "/"
	case TypeModulus:
		return "%"
	case TypeExponentiation:
		return "^"
	case TypeLogicalEqual:
		return "=="
	case TypeLogicalNotEqual:
		return "!="
	case TypeEqual:
		return "="
	case TypeNotEqual:
		return "<>"
	case TypeGreaterThan:
		return ">"
	case TypeGreaterThanEqual:
		return ">="
	case TypeLessThan:
		return "<"
	case TypeLessThanEqual:
		return "<="
	case TypeTrue:
		return "true"
	case TypeFalse:
		return "false"
	case TypeNot:
		return "not"
	case TypeAnd:
		return "and"
	case TypeOr:
		return "or"
	case TypeTimeEqual:
		return "t == begin"
	case TypeIdentifierSpecific:
		temp := fmt.Sprintf("Specific %v:", code.Int)
		for _, c := range code.IndexOp {
			temp += c.String() + " "
		}
		return code.Str + temp
	case TypeIdentifierGeneral:
		temp := "General: "
		for _, c := range code.IndexOp {
			temp += c.String() + " "
		}
		return code.Str + temp
	case TypeIdentifierThis:
		temp := "This: "
		for _, c := range code.IndexOp {
			temp += c.String() + " "
		}
		return code.Str + temp
	case TypeIdentifierSpecificRange:
		temp := fmt.Sprintf("Specific Range %v:", code.Int)
		for _, c := range code.IndexOp {
			temp += c.String() + " "
		}
		return code.Str + temp
	case TypeIdentifierGeneralRange:
		temp := "General Range: "
		for _, c := range code.IndexOp {
			temp += c.String() + " "
		}
		return code.Str + temp
	case TypeIdentifierThisRange:
		temp := "This Range: "
		for _, c := range code.IndexOp {
			temp += c.String() + " "
		}
		return code.Str + temp
	case TypeThen:
		return "then"
	case TypeElse:
		return "else"
	case TypeFunctionCall:
		return "Function Call " + code.Str
	case TypeString:
		return "String " + code.Str
	case TypeIdentifierCategory:
		return "Category"
	case TypeStringEqual:
		return "== (string)"
	case TypeStringNotEqual:
		return "!= (string)"
	}
	return "Unknown Type"
}

type Expression struct {
	Code                []ByteCode
	Top                 int
	CurrentFunctionName string
}

func (e *Expression) IsAppliedOverAllSeries() bool {
	for _, code := range e.Code[0:e.Top] {
		if code.T == TypeIdentifierGeneral {
			return true
		}
	}

	return false
}

func (e *Expression) IsTimeVarying() bool {
	for _, code := range e.Code[0:e.Top] {
		if code.IndexOp != nil {
			for _, indexOp := range code.IndexOp {
				if indexOp.T == TypeCurrentTime {
					return true
				}
			}
		} else if code.T == TypeIdentifierGeneral || code.T == TypeIdentifierSpecific || code.T == TypeIdentifierCategory {
			// We have a formula of type val or val1 or a category
			return true
		}
	}

	return false
}

func (e *Expression) IsComputeAcrossSeries() (bool, int) {
	seriesNums := make([]int, 0)

	for _, code := range e.Code[0:e.Top] {
		if code.T == TypeIdentifierGeneral {
			seriesNums = append(seriesNums, -1)
		} else if code.T == TypeIdentifierSpecific {
			seriesNums = append(seriesNums, code.Int)
		}
	}

	isComputeAcrossSeries := !intIsUnique(seriesNums)
	if !isComputeAcrossSeries && len(seriesNums) > 0 {
		return isComputeAcrossSeries, seriesNums[0]
	}
	return isComputeAcrossSeries, -1
}

func intIsUnique(arr []int) bool {
	if arr == nil || len(arr) == 0 {
		return true
	}

	sort.Ints(arr)

	if len(arr) < 2 {
		return true
	}

	for i := 1; i < len(arr); i++ {
		if arr[i] != arr[i-1] {
			return false
		}
	}

	return true
}

func (e *Expression) Init(expression string) {
	e.Code = make([]ByteCode, len(expression))
}

func (e *Expression) AddFunctionArgument() {
	// Do nothing for now
}

func (e *Expression) AddIndexOperator(operator Type) {
	if e.Code[e.Top-1].IndexOp == nil {
		e.Code[e.Top-1].IndexOp = make([]IndexCode, 0)
	}
	e.Code[e.Top-1].IndexOp = append(e.Code[e.Top-1].IndexOp, IndexCode{T: operator})
}

func (e *Expression) AddIndexValue(value string) {
	i, _ := strconv.ParseInt(value, 10, 64)

	if e.Code[e.Top-1].IndexOp == nil {
		e.Code[e.Top-1].IndexOp = make([]IndexCode, 0)
	}
	e.Code[e.Top-1].IndexOp = append(e.Code[e.Top-1].IndexOp, IndexCode{T: TypeNumber, Int: int(i)})
}

func (e *Expression) AddOperator(operator Type) {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = operator
}

func (e *Expression) AddFunctionName(name string) {
	e.CurrentFunctionName = name
}

func (e *Expression) AddFunctionCall() {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeFunctionCall
	code[top].Str = e.CurrentFunctionName
}

func (e *Expression) AddIdentifierSpecific(value string) {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeIdentifierSpecific

	i, _ := strconv.ParseInt(value, 10, 64)

	code[top].Int = int(i - 1)
}

func (e *Expression) AddIdentifierGeneral() {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeIdentifierGeneral
}

func (e *Expression) AddIdentifierThis() {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeIdentifierThis
}

func (e *Expression) AddIdentifierSpecificRange(value string) {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeIdentifierSpecificRange

	i, _ := strconv.ParseInt(value, 10, 64)

	code[top].Int = int(i - 1)
}

func (e *Expression) AddIdentifierGeneralRange() {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeIdentifierGeneralRange
}

func (e *Expression) AddIdentifierThisRange() {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeIdentifierThisRange
}

func (e *Expression) AddCategoryIdentifier() {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeIdentifierCategory
}

func (e *Expression) AddValue(value string) {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeNumber
	code[top].Float, _ = strconv.ParseFloat(value, 64)
}

func (e *Expression) AddStringValue(value string) {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeString
	code[top].Str = value
}

func (e *Expression) String() string {
	s := ""

	for _, code := range e.Code[0:e.Top] {
		s += code.String() + "\n"
	}

	return s
}

func (e *Expression) Evaluate() float64 {
	val := []float64{1, 2, 3, 4, 5}
	category := "Hello World"
	length := len(val)
	currentIndex := 2

	stack, top := make([]float64, len(e.Code)), 0
	booleanStack := make([]bool, len(e.Code))
	arrayStack, arrayStackTop := make([][]float64, len(e.Code)), 0

	var string1, string2 string
	for i := 0; i < e.Top; i++ {
		code := e.Code[i]
		fmt.Printf("%s: i=%v, top=%v, stack=%v, booleanStack=%v, arrayStack=%v\n", code.String(), i, top, stack[0:top], booleanStack[0:top], arrayStack[0:arrayStackTop])

		// These operators add to the stack
		switch code.T {
		case TypeNumber:
			stack[top] = code.Float
			top++
			continue
		case TypeString:
			if string1 == "" {
				string1 = code.Str
			} else {
				string2 = code.Str
			}
			continue
		case TypeIdentifierCategory:
			if string1 == "" {
				string1 = category
			} else {
				string2 = category
			}
			continue
		case TypeIdentifierSpecific:
			// This will access a specific series
			idx, _, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				stack[top] = val[idx]
			} else {
				fmt.Printf("err=%s\n", err)
			}
			top++
			continue
		case TypeIdentifierGeneral:
			// This will access the *current* series (similar to the *current* time)
			idx, _, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				stack[top] = val[idx]
			} else {
				fmt.Printf("err=%s\n", err)
			}
			top++
			continue
		case TypeIdentifierThis:
			// This will access the *current* series (similar to the *current* time)
			idx, _, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				stack[top] = 2048 + val[idx] // TODO: Change to the current computed array
			} else {
				fmt.Printf("err=%s\n", err)
			}
			top++
			continue
		case TypeIdentifierSpecificRange:
			// This will access a specific series
			idx1, idx2, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				// fmt.Printf(" idx1=%v, idx2=%v ", idx1, idx2)
				arrayStack[arrayStackTop] = val[idx1:(1 + idx2)]
			} else {
				fmt.Printf("err=%s\n", err)
			}
			arrayStackTop++
			continue
		case TypeIdentifierGeneralRange:
			// This will access the *current* series (similar to the *current* time)
			idx1, idx2, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				// fmt.Printf(" idx1=%v, idx2=%v ", idx1, idx2)
				arrayStack[arrayStackTop] = val[idx1:(1 + idx2)]
			} else {
				fmt.Printf("err=%s\n", err)
			}
			arrayStackTop++
			continue
		case TypeIdentifierThisRange:
			// This will access the *computed* series
			idx1, idx2, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				arrayStack[arrayStackTop] = val[idx1:(1 + idx2)] // TODO: Change this to the current computed array
			} else {
				fmt.Printf("err=%s\n", err)
			}
			arrayStackTop++
			continue
		case TypeNegation:
			stack[top-1] = -1 * stack[top-1]
			continue
		case TypeTrue:
			booleanStack[top] = true
			top++
			continue
		case TypeFalse:
			booleanStack[top] = false
			top++
			continue
		case TypeNot:
			booleanStack[top-1] = !booleanStack[top-1]
			continue
		case TypeStringEqual:
			booleanStack[top] = strings.ToLower(string1) == strings.ToLower(string2)
			string1, string2 = "", ""
			top++
			continue
		case TypeStringNotEqual:
			booleanStack[top] = strings.ToLower(string1) != strings.ToLower(string2)
			string1, string2 = "", ""
			top++
			continue
		case TypeTimeEqual:
			booleanStack[top] = currentIndex == 0
			top++
			continue
		}

		// These operators remove from the stack
		switch code.T {
		case TypeAdd:
			stack[top-2] = stack[top-2] + stack[top-1]
		case TypeSubtract:
			stack[top-2] = stack[top-2] - stack[top-1]
		case TypeMultiply:
			stack[top-2] = stack[top-2] * stack[top-1]
		case TypeDivide:
			stack[top-2] = stack[top-2] / stack[top-1]
		case TypeModulus:
			stack[top-2] = math.Mod(stack[top-2], stack[top-1])
		case TypeExponentiation:
			stack[top-2] = math.Pow(stack[top-2], stack[top-1])
		case TypeLogicalEqual:
			booleanStack[top-2] = booleanStack[top-2] == booleanStack[top-1]
		case TypeLogicalNotEqual:
			booleanStack[top-2] = booleanStack[top-2] != booleanStack[top-1]
		case TypeAnd:
			booleanStack[top-2] = booleanStack[top-2] && booleanStack[top-1]
		case TypeOr:
			booleanStack[top-2] = booleanStack[top-2] || booleanStack[top-1]
		case TypeEqual:
			booleanStack[top-2] = stack[top-2] == stack[top-1]
		case TypeNotEqual:
			booleanStack[top-2] = stack[top-2] != stack[top-1]
		case TypeGreaterThan:
			booleanStack[top-2] = stack[top-2] > stack[top-1]
		case TypeGreaterThanEqual:
			booleanStack[top-2] = stack[top-2] >= stack[top-1]
		case TypeLessThan:
			booleanStack[top-2] = stack[top-2] < stack[top-1]
		case TypeLessThanEqual:
			booleanStack[top-2] = stack[top-2] <= stack[top-1]
		case TypeThen:
			if booleanStack[0] {
				// Continue on to the next operation
			} else {
				// Branch to the else
				i = e.FindElseAfter(i)
			}
		case TypeElse:
			return stack[0]
		case TypeFunctionCall:
			return 1024.0
		}
		top--
	}
	return stack[0]
}

func (e Expression) FindElseAfter(i int) int {
	for j := i; j < e.Top; j++ {
		if e.Code[j].T == TypeElse {
			return j
		}
	}

	return i
}

func (e *ByteCode) EvaluateIndexString() (string, error) {
	stack, top := make([]string, len(e.IndexOp)), 0

	if len(stack) == 0 {
		return "", nil
	}

	for _, code := range e.IndexOp {
		switch code.T {
		case TypeNumber:
			stack[top] = fmt.Sprintf("%v", code.Int)
			top++
			continue
		case TypeBegin:
			stack[top] = "begin"
			top++
			continue
		case TypeEnd:
			stack[top] = "end"
			top++
			continue
		case TypeCurrentTime:
			stack[top] = "t"
			top++
			continue
		case TypeNegation:
			stack[top-1] = "-" + stack[top-1]
			continue
		}

		switch code.T {
		case TypeAdd:
			stack[top-2] = stack[top-2] + "+" + stack[top-1]
		case TypeSubtract:
			stack[top-2] = stack[top-2] + "-" + stack[top-1]
		case TypeTimeRange:
			stack[top-2] = stack[top-2] + ":" + stack[top-1]
		}
		top--
	}

	if stack[0] != "" {
		stack[0] = "[" + stack[0] + "]"
	}

	return stack[0], nil
}

func (e *ByteCode) EvaluateIndex(currentIndex, length int) (int, int, error) {
	stack, top := make([]int, len(e.IndexOp)), 0

	if len(stack) == 0 {
		if 0 <= currentIndex && currentIndex < length {
			return currentIndex, currentIndex, nil
		}
		return 0, 0, errors.New("out of bounds")
	}

	for _, code := range e.IndexOp {
		switch code.T {
		case TypeNumber:
			stack[top] = code.Int
			top++
			continue
		case TypeBegin:
			stack[top] = 0
			top++
			continue
		case TypeEnd:
			if length > 0 {
				stack[top] = length - 1
			} else {
				stack[top] = 0
			}
			top++
			continue
		case TypeCurrentTime:
			stack[top] = currentIndex
			top++
			continue
		case TypeNegation:
			stack[top-1] = -1 * stack[top-1]
			continue
		}

		switch code.T {
		case TypeAdd:
			stack[top-2] = stack[top-2] + stack[top-1]
		case TypeSubtract:
			stack[top-2] = stack[top-2] - stack[top-1]
		case TypeTimeRange:
			// Bounds checks
			if 0 <= stack[top-1] && stack[top-1] < length && 0 <= stack[top-2] && stack[top-2] < length && stack[top-2] <= stack[top-1] {
				return stack[top-2], stack[top-1], nil
			}
			return 0, 0, errors.New("out of bounds")
		}
		top--
	}

	idx := stack[0]

	// Bounds checks
	if 0 <= idx && idx < length {
		return idx, idx, nil
	}
	return 0, 0, errors.New("out of bounds")
}

func ParseHandler(w http.ResponseWriter, r *http.Request, expression string) {
	var calc *Calculator
	calc = &Calculator{Buffer: expression}
	calc.Init()
	calc.Expression.Init(expression)
	if err := calc.Parse(); err != nil {
		fmt.Fprintf(w, `{"status": "err"}`)
	} else {
		fmt.Fprintf(w, `{"status": "ok"}`)
	}
}
