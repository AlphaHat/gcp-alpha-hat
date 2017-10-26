package parse

import "log"

var calc *Calculator

func ExampleParse() {
	//expression := "val[t-1] + 5 % 2 + 3.5 + val0 + val[begin] + val * val[end] + val[end-1] + val1[t-1] * 1024"
	//expression := "if 1>0 and (2==(3+5))=false then 5 else if 5 < 6 then 8+9 else 9*12*24"
	//expression := "if 5 < 6 then 5 else 6"
	//expression := "val0[t-1]"
	//expression := "val0[t-2] + val[t-1]"
	//expression := "if 1>0 then if 2 > 3 then 6 else 7 else 0"
	// expression := "if val[t+1]/val[t]-1>0.03 or 1=1 then val[t] else 0"
	//expression := "1 + val + fn(0, 1, val1)"
	//expression := `if category=="Hello world" then 1 else 0`
	// expression := "1 + sum(val[t-1:t])"
	// expression := "val[t] + sum(this[t-1:t])"
	// expression := "if t == begin then 0 else 1"
	expression := "val"
	calc = &Calculator{Buffer: expression}
	calc.Init()
	calc.Expression.Init(expression)
	if err := calc.Parse(); err != nil {
		log.Fatal(err)
	}
	// calc.Execute()
	// fmt.Printf("%s =\n%s\n", expression, calc.String())

	// Output:
	//
}

//
// func ExampleExecute() {
// 	fmt.Printf("= %v\n", calc.Evaluate())
//
// 	// Output:
// 	//
// }
