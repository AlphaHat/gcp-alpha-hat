package run

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/AlphaHat/gcp-alpha-hat/parse"
)

func convertExpressionToFunction(expression string, label string, e *parse.Expression) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		// Check if there's a SameEntityAggregation (i.e. two different TypeIdentifierSpecific in the formula)
		// If so, we'll have to force a time series alignment
		isComputeAcrossSeries, specificFormulaIndex := e.IsComputeAcrossSeries()
		if isComputeAcrossSeries {
			s.Data = forceSeriesAlignment(s.Data)
		}

		var numSeries int

		// Check if there's a TypeIdentifierGeneral
		if e.IsAppliedOverAllSeries() {
			numSeries = len(s.Data)
		} else {
			if len(s.Data) > 0 {
				numSeries = 1
			} else {
				numSeries = 0
			}
		}

		newSeries := make([]Series, numSeries)

		for seriesNum := 0; seriesNum < numSeries; seriesNum++ {
			// If this formula refers to a specific series only, we should use the dates and data from that series
			var inputSeriesNum int
			if specificFormulaIndex >= 0 {
				inputSeriesNum = specificFormulaIndex
			} else {
				inputSeriesNum = seriesNum
			}

			//newSeries[seriesNum].IsWeight = s.Data[seriesNum].IsWeight
			newSeries[seriesNum].IsWeight = strings.ToLower(label) == "weight"
			if inputSeriesNum >= len(s.Data) {
				return s
			}
			newSeries[seriesNum].Meta = s.Data[inputSeriesNum].Meta
			if label == "" {
				newSeries[seriesNum].Meta.Label, _ = evaluateLabel(e, &s, inputSeriesNum)
			} else if e.IsAppliedOverAllSeries() {
				newSeries[seriesNum].Meta.Label += (" " + label)
			} else {
				newSeries[seriesNum].Meta.Label = label
			}
			newSeries[seriesNum].Meta.Units, _ = evaluateUnits(e, &s, inputSeriesNum)

			var numTimePoints int

			// Check if the formula is time-varying. Should check for either no IndexOps or no TypeCurrentTime
			// If it's not time-varying, the output should only be on the last day.
			// If you wanna display the same value each day, the hack would be to say +val[t]-val[t]
			if e.IsTimeVarying() {
				numTimePoints = len(s.Data[inputSeriesNum].Data)
			} else {
				if len(s.Data[inputSeriesNum].Data) > 0 {
					numTimePoints = 1
				} else {
					numTimePoints = 0
				}
			}

			for t := 0; t < numTimePoints; t++ {
				data, err := evaluateExpression(e, &s, newSeries[seriesNum].Data, inputSeriesNum, t)

				if err == nil {
					var currentTime time.Time
					if e.IsTimeVarying() {
						currentTime = s.Data[inputSeriesNum].Data[t].Time
					} else {
						currentTime = s.Data[inputSeriesNum].Data[len(s.Data[inputSeriesNum].Data)-1].Time
					}
					newSeries[seriesNum].Data = append(newSeries[seriesNum].Data, DataPoint{Time: currentTime, Data: data})
				} else {
				}
			}

		}

		s.Data = append(s.Data, newSeries...)

		s.Data = removeNaNs(s.Data)

		return s
	}
}

func evaluateLabel(e *parse.Expression, s *SingleEntityData, seriesNum int) (string, error) {
	stack, top := make([]string, len(e.Code)), 0
	for _, code := range e.Code[0:e.Top] {
		switch code.T {
		case parse.TypeNumber:
			stack[top] = fmt.Sprintf("%v", code.Float)
			top++
			continue
		case parse.TypeString:
			continue
		case parse.TypeIdentifierCategory:
			continue
		case parse.TypeIdentifierSpecificRange, parse.TypeIdentifierSpecific:
			// This will access a specific series
			if code.Int >= len(s.Data) {
				return "", errors.New("index out bounds")
			}
			idx, _ := code.EvaluateIndexString()
			stack[top] = s.Data[code.Int].Meta.Label + idx
			top++
			continue
		case parse.TypeIdentifierGeneralRange, parse.TypeIdentifierGeneral:
			// This will access the *current* series (similar to the *current* time)
			if seriesNum >= len(s.Data) {
				return "", errors.New("index out bounds")
			}
			idx, _ := code.EvaluateIndexString()
			stack[top] = s.Data[seriesNum].Meta.Label + idx
			top++
			continue
		case parse.TypeIdentifierThis, parse.TypeIdentifierThisRange:
			top++
			continue
		case parse.TypeNegation:
			continue
		case parse.TypeTrue:
			top++
			continue
		case parse.TypeFalse:
			top++
			continue
		case parse.TypeNot:
			continue
		case parse.TypeStringEqual:
			top++
			continue
		case parse.TypeStringNotEqual:
			top++
			continue
		case parse.TypeTimeEqual:
			top++
			continue
		}

		switch code.T {
		case parse.TypeAdd:
			stack[top-2] = stack[top-2] + "+" + stack[top-1]
		case parse.TypeSubtract:
			stack[top-2] = stack[top-2] + "-" + stack[top-1]
		case parse.TypeMultiply:
			stack[top-2] = stack[top-2] + "*" + stack[top-1]
		case parse.TypeDivide:
			stack[top-2] = stack[top-2] + "/" + stack[top-1]
		case parse.TypeModulus:
			stack[top-2] = stack[top-2] + "%" + stack[top-1]
		case parse.TypeExponentiation:
			stack[top-2] = stack[top-2] + "^" + stack[top-1]
		case parse.TypeElse:
			return stack[0], nil
		case parse.TypeFunctionCall:
			stack[top] = code.Str + "(" + stack[top-1] + ")"
			top++
			continue
		}
		top--
	}
	return stack[0], nil
}

func evaluateUnits(e *parse.Expression, s *SingleEntityData, seriesNum int) (string, error) {
	stack, top := make([]string, len(e.Code)), 0
	for _, code := range e.Code[0:e.Top] {
		switch code.T {
		case parse.TypeNumber:
			stack[top] = "Constant"
			top++
			continue
		case parse.TypeString:
			continue
		case parse.TypeIdentifierCategory:
			continue
		case parse.TypeIdentifierSpecificRange, parse.TypeIdentifierSpecific:
			// This will access a specific series
			if code.Int >= len(s.Data) {
				return "", errors.New("index out bounds")
			}
			stack[top] = s.Data[code.Int].Meta.Units
			top++
			continue
		case parse.TypeIdentifierGeneralRange, parse.TypeIdentifierGeneral:
			// This will access the *current* series (similar to the *current* time)
			if seriesNum >= len(s.Data) {
				return "", errors.New("index out bounds")
			}
			stack[top] = s.Data[seriesNum].Meta.Units
			top++
			continue
		case parse.TypeIdentifierThis, parse.TypeIdentifierThisRange:
			top++
			continue
		case parse.TypeNegation:
			continue
		case parse.TypeTrue:
			top++
			continue
		case parse.TypeFalse:
			top++
			continue
		case parse.TypeNot:
			continue
		case parse.TypeStringEqual:
			top++
			continue
		case parse.TypeStringNotEqual:
			top++
			continue
		case parse.TypeTimeEqual:
			top++
			continue
		}

		switch code.T {
		case parse.TypeAdd:
			//stack[top-2] = stack[top-2] + stack[top-1]
		case parse.TypeSubtract:
			if stack[top-1] == "Constant" && stack[top-2] == "Ratio" {
				stack[top-2] = "%"
			}
		case parse.TypeMultiply:
			if stack[top-1] != "Constant" {
				stack[top-2] = stack[top-2] + "*" + stack[top-1]
			}
		case parse.TypeDivide:
			if stack[top-2] == stack[top-1] {
				stack[top-2] = "Ratio"
			} else if stack[top-1] != "Constant" && stack[top-2] != "Constant" {
				stack[top-2] = stack[top-2] + "/" + stack[top-1]
			}
		case parse.TypeModulus:
		case parse.TypeExponentiation:
		case parse.TypeLogicalEqual:
		case parse.TypeLogicalNotEqual:
		case parse.TypeAnd:
		case parse.TypeOr:
		case parse.TypeEqual:
		case parse.TypeNotEqual:
		case parse.TypeGreaterThan:
		case parse.TypeGreaterThanEqual:
		case parse.TypeLessThan:
		case parse.TypeLessThanEqual:
		case parse.TypeThen:
			// Continue on to the next operation
		case parse.TypeElse:
			return stack[0], nil
		case parse.TypeFunctionCall:
			functionName := code.Str
			valence := getValence(functionName)
			top = top - valence
			if valence == 2 {
				stack[top] = functionUnits2(functionName, stack[top], stack[top+1])
			} else {
				stack[top] = functionUnits1(functionName, stack[top])
			}
			top++
			continue
		}
		top--
	}
	return stack[0], nil
}

func evaluateExpression(e *parse.Expression, s *SingleEntityData, this []DataPoint, seriesNum int, currentIndex int) (float64, error) {
	stack, top := make([]float64, len(e.Code)), 0
	booleanStack := make([]bool, len(e.Code))
	arrayStack, arrayStackTop := make([][]DataPoint, len(e.Code)), 0

	var string1, string2 string
	for i := 0; i < e.Top; i++ {
		code := e.Code[i]
		switch code.T {
		case parse.TypeNumber:
			stack[top] = code.Float
			top++
			continue
		case parse.TypeString:
			if string1 == "" {
				string1 = strings.TrimSpace(code.Str)
			} else {
				string2 = strings.TrimSpace(code.Str)
			}
			continue
		case parse.TypeIdentifierCategory:
			if seriesNum >= len(s.Data) {
				return 0, errors.New("index out bounds")
			}
			val := s.Data[seriesNum].Data
			length := len(val)
			idx, _, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				t := val[idx].Time // Series identified by seriesNum
				category := s.Category.LookupCategory(t)

				if string1 == "" {
					string1 = strings.TrimSpace(category)
				} else {
					string2 = strings.TrimSpace(category)
				}
			}

			continue
		case parse.TypeIdentifierSpecific:
			// This will access a specific series
			if code.Int >= len(s.Data) {
				return 0, errors.New("index out bounds")
			}
			val := s.Data[code.Int].Data
			length := len(val)
			idx, _, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				stack[top] = val[idx].Data // Specific series identified by code.Int
			} else {
				return 0, err
			}
			top++
			continue
		case parse.TypeIdentifierGeneral:
			// This will access the *current* series (similar to the *current* time)
			if seriesNum >= len(s.Data) {
				return 0, errors.New("index out bounds")
			}
			val := s.Data[seriesNum].Data
			length := len(val)
			idx, _, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				stack[top] = val[idx].Data // Series identified by seriesNum
			} else {
				return 0, err
			}
			top++
			continue
		case parse.TypeIdentifierThis:
			// This will access the *current* series (similar to the *current* time)
			val := this
			length := len(val)
			idx, _, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				stack[top] = val[idx].Data // Series identified by seriesNum
			} else {
				return 0, err
			}
			top++
			continue
		case parse.TypeIdentifierSpecificRange:
			// This will access a specific series
			if code.Int >= len(s.Data) {
				return 0, errors.New("index out bounds")
			}
			val := s.Data[code.Int].Data
			length := len(val)

			idx1, idx2, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				// fmt.Printf(" idx1=%v, idx2=%v ", idx1, idx2)
				arrayStack[arrayStackTop] = val[idx1:(1 + idx2)]
			} else {
				return 0, err
			}
			arrayStackTop++
			continue
		case parse.TypeIdentifierGeneralRange:
			// This will access the *current* series (similar to the *current* time)
			if seriesNum >= len(s.Data) {
				return 0, errors.New("index out bounds")
			}
			val := s.Data[seriesNum].Data
			length := len(val)

			idx1, idx2, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				// fmt.Printf(" idx1=%v, idx2=%v ", idx1, idx2)
				arrayStack[arrayStackTop] = val[idx1:(1 + idx2)]
			} else {
				return 0, err
			}
			arrayStackTop++
			continue
		case parse.TypeIdentifierThisRange:
			// This will access the *computed* series
			val := this
			length := len(val)

			idx1, idx2, err := code.EvaluateIndex(currentIndex, length)
			if err == nil {
				arrayStack[arrayStackTop] = val[idx1:(1 + idx2)]
			} else {
				return 0, err
			}
			arrayStackTop++
			continue
		case parse.TypeNegation:
			stack[top-1] = -1 * stack[top-1]
			continue
		case parse.TypeTrue:
			booleanStack[top] = true
			top++
			continue
		case parse.TypeFalse:
			booleanStack[top] = false
			top++
			continue
		case parse.TypeNot:
			booleanStack[top-1] = !booleanStack[top-1]
			continue
		case parse.TypeStringEqual:
			booleanStack[top] = strings.ToLower(string1) == strings.ToLower(string2)
			string1, string2 = "", ""
			top++
			continue
		case parse.TypeStringNotEqual:
			booleanStack[top] = strings.ToLower(string1) != strings.ToLower(string2)
			string1, string2 = "", ""
			top++
			continue
		case parse.TypeTimeEqual:
			booleanStack[top] = currentIndex == 0
			top++
			continue
		}

		switch code.T {
		case parse.TypeAdd:
			stack[top-2] = stack[top-2] + stack[top-1]
		case parse.TypeSubtract:
			stack[top-2] = stack[top-2] - stack[top-1]
		case parse.TypeMultiply:
			stack[top-2] = stack[top-2] * stack[top-1]
		case parse.TypeDivide:
			stack[top-2] = stack[top-2] / stack[top-1]
		case parse.TypeModulus:
			stack[top-2] = math.Mod(stack[top-2], stack[top-1])
		case parse.TypeExponentiation:
			stack[top-2] = math.Pow(stack[top-2], stack[top-1])
		case parse.TypeLogicalEqual:
			booleanStack[top-2] = booleanStack[top-2] == booleanStack[top-1]
		case parse.TypeLogicalNotEqual:
			booleanStack[top-2] = booleanStack[top-2] != booleanStack[top-1]
		case parse.TypeAnd:
			booleanStack[top-2] = booleanStack[top-2] && booleanStack[top-1]
		case parse.TypeOr:
			booleanStack[top-2] = booleanStack[top-2] || booleanStack[top-1]
		case parse.TypeEqual:
			booleanStack[top-2] = stack[top-2] == stack[top-1]
		case parse.TypeNotEqual:
			booleanStack[top-2] = stack[top-2] != stack[top-1]
		case parse.TypeGreaterThan:
			booleanStack[top-2] = stack[top-2] > stack[top-1]
		case parse.TypeGreaterThanEqual:
			booleanStack[top-2] = stack[top-2] >= stack[top-1]
		case parse.TypeLessThan:
			booleanStack[top-2] = stack[top-2] < stack[top-1]
		case parse.TypeLessThanEqual:
			booleanStack[top-2] = stack[top-2] <= stack[top-1]
		case parse.TypeThen:
			if booleanStack[0] {
				// Continue on to the next operation
			} else {
				// Branch to the else
				i = e.FindElseAfter(i)
			}
		case parse.TypeElse:
			return stack[0], nil
		case parse.TypeFunctionCall:
			// Identify valence of the function
			// Send variadic arguments to the function
			functionName := code.Str
			valence := getValence(functionName)
			arrayStackTop = arrayStackTop - valence
			if valence == 2 {
				stack[top] = runFunction2(functionName, arrayStack[arrayStackTop], arrayStack[arrayStackTop+1])
			} else {
				stack[top] = runFunction1(functionName, arrayStack[arrayStackTop])
			}
			top++
			continue
		}
		top--
	}
	return stack[0], nil
}

func parseTimeSeriesTransformation(expression string) (*parse.Expression, error) {
	calc := &parse.Calculator{Buffer: strings.ToLower(expression)}
	calc.Init()
	calc.Expression.Init(expression)

	err := calc.Parse()

	if err == nil {
		calc.Execute()
		return &calc.Expression, nil
	}
	return nil, err
}

func formulaToFunction(expression string, label string) func(SingleEntityData) SingleEntityData {
	e, err := parseTimeSeriesTransformation(expression)

	if err == nil && e != nil {
		fn := convertExpressionToFunction(expression, label, e)
		return fn
	} else {
		// zlog.LogError("formulaToFunction", err)
	}

	return func(s SingleEntityData) SingleEntityData {
		return s
	}
}
