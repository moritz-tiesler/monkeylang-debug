package driver

import (
	"monkey/compiler"
	"monkey/parser"
	"monkey/vm"
	"testing"
)

type driverTestCase struct {
	sourceCode       string
	breakPoints      []breakpoint
	expectedHits     []breakpoint
	expectedLocation int
}

func TestRunUntilBreakPoint(t *testing.T) {
	tests := []driverTestCase{
		{
			sourceCode: `let square = fn(x) {
	return x * x
}
let squareAndDouble = fn(a) {
	let b = square(a) * 2
	return b
}
let z = square(2)
puts(z)
let y = squareAndDouble(2)`,

			breakPoints: []breakpoint{
				{line: 2, col: 0},
				{line: 10, col: 0},
			},
		},
		{
			sourceCode: `let square = fn(x) {
	return x 
}
let z = 3
let y = 4`,

			breakPoints: []breakpoint{
				{line: 4, col: 0},
			},
		},
	}

	var err error
	var bpHit bool
	for i, tt := range tests {
		driver := New()
		err = driver.StartVM(tt.sourceCode)
		if err != nil {
			t.Errorf("error starting VM: %s", err)
		}
		for _, bp := range tt.breakPoints {
			err, bpHit = driver.RunUntilBreakPoint(bp.line)
			if err != nil {
				t.Errorf("error running driver")
			}
		}

		if !bpHit {
			t.Errorf("error test=%d: expected to hit breakpoint, hit none", i+1)
		} else {
			expected := tt.breakPoints[len(tt.breakPoints)-1].line
			vmLoc := driver.VM.SourceLocation()
			actual := vmLoc.Range.Start.Line
			if expected != actual {
				t.Errorf("error in breaktpoint test %d", i)
				t.Errorf("wrong breakpoint line: expected line=%d, got line=%d", expected, actual)
			}
		}
	}
}

func TestRunWithBreakpoints(t *testing.T) {
	tests := []driverTestCase{
		{
			sourceCode: `let square = fn(x) {
	let res = x
	return res
}
let squareAndDouble = fn(a) {
	let b = square(a)
	return b
}
let z = square(2)
let y = squareAndDouble(2)
let q = y
let bb = y
let bb = y`,

			breakPoints: []breakpoint{
				{line: 2, col: 0},
				{line: 12, col: 0},
			},
			expectedHits: []breakpoint{
				{line: 2, col: 0},
				{line: 2, col: 0},
				{line: 12, col: 0},
			},
		},
	}

	for i, tt := range tests {
		driver := New()
		err := driver.StartVM(tt.sourceCode)
		driver.SourceCode = tt.sourceCode
		if err != nil {
			t.Errorf("error starting VM: %s", err)
		}

		actualHits := []breakpoint{}
		for j := 0; j < len(tt.expectedHits); j++ {
			err, bpHit := driver.RunWithBreakpoints(tt.breakPoints)
			//t.Logf(driver.State())
			if err != nil {
				t.Errorf("%s\n", err)
			}
			if bpHit {
				vmLoc := driver.VM.SourceLocation()
				vmHit := vmLoc.Range.Start.Line
				t.Logf("vmHit=%v", vmHit)
				actualHits = append(actualHits, breakpoint{line: vmHit})
			}
		}
		for k, bp := range tt.expectedHits {
			expected := bp.line
			actual := actualHits[k].line
			if expected != actual {
				t.Errorf("wrong breaktpoint in test %d", i)
				t.Errorf("wrong breakpoint line: expected=%d, got=%d", expected, actual)
			}
		}

	}
}

func TestStepOver(t *testing.T) {
	tests := []driverTestCase{
		{
			sourceCode: `
let square = fn(x) {
	return x * x
}
let squareAndDouble = fn(a) {
	let b = square(a) * 2
	b
}
let res = squareAndDouble(2)`,

			breakPoints: []breakpoint{
				{line: 6, col: 0},
			},
			expectedLocation: 7,
		}, {
			sourceCode: `
let func = fn(a) {a}
let res = func(4)
let res = func(4)`,

			breakPoints: []breakpoint{
				{line: 3, col: 0},
			},
			expectedLocation: 4,
		},
		{
			sourceCode: `
let func = fn(a) {a}
let res =func(4)
let res =func(4)`,

			breakPoints: []breakpoint{
				{line: 4, col: 0},
			},
			expectedLocation: 0,
		},
		{
			sourceCode: `
let func = fn(a) {a}
let res =func(4)
let res =func(4)`,

			breakPoints:      []breakpoint{},
			expectedLocation: 2,
		},
		{
			sourceCode: `let square = fn(x) {
	return x * x
}
let squareAndDouble = fn(a) {
	let b = square(a) * 2
	return b
}
let z = square(2)
let y = squareAndDouble(2)
let bogus = 3`,

			breakPoints: []breakpoint{
				{line: 2, col: 0},
			},
			expectedLocation: 8,
		},
		{
			sourceCode: `
let x = 2
x`,

			breakPoints: []breakpoint{
				{line: 2, col: 0},
			},
			expectedLocation: 3,
		},
		{
			sourceCode: `
let x = 2
x`,

			breakPoints: []breakpoint{
				{line: 3, col: 0},
			},
			expectedLocation: 0,
		},
	}

	for i, tt := range tests {
		driver := New()
		err := driver.StartVM(tt.sourceCode)
		if err != nil {
			t.Errorf("error starting VM: %s", err)
		}
		driver.Breakpoints = tt.breakPoints
		driver.SourceCode = tt.sourceCode
		for _, bp := range tt.breakPoints {
			driver.RunUntilBreakPoint(bp.line)
			driver.StepOver()
			t.Logf(driver.State())
			t.Logf("%d", driver.VM.State())
		}

		expected := tt.expectedLocation
		vmLoc := driver.VM.SourceLocation()
		actual := vmLoc.Range.Start.Line
		if expected != actual {
			t.Errorf("error in breaktpoint test %d", i+1)
			t.Errorf("wrong breakpoint line: expected line=%d, got line=%d", expected, actual)
		}
	}
}

func TestStepInto(t *testing.T) {
	tests := []driverTestCase{
		{
			sourceCode: `
let func = fn(a) {a}
let res = func(4)
let res = func(4)`,

			breakPoints: []breakpoint{
				{line: 3, col: 0},
			},
			expectedLocation: 2,
		},
		{
			sourceCode: `
let func = fn(a) {a}
let res =func(4)
let res =func(4)`,

			breakPoints: []breakpoint{
				{line: 4, col: 0},
			},
			expectedLocation: 2,
		},
		{
			sourceCode: `
let square = fn(x) {
	return x * x
}
let squareAndDouble = fn(a) {
	let b = square(a) * 2
	return b
}
let z = square(2)
let y = squareAndDouble(2)
let bogus = 3`,

			breakPoints: []breakpoint{
				{line: 6, col: 0},
			},
			expectedLocation: 3,
		},
		{
			sourceCode: `
let x = 2
x`,

			breakPoints: []breakpoint{
				{line: 2, col: 0},
			},
			expectedLocation: 3,
		},
		{
			sourceCode: `
let x = 2
x`,

			breakPoints: []breakpoint{
				{line: 3, col: 0},
			},
			expectedLocation: 0,
		},
	}

	for i, tt := range tests {
		driver := New()
		err := driver.StartVM(tt.sourceCode)
		if err != nil {
			t.Errorf("error starting VM: %s", err)
		}
		driver.Breakpoints = tt.breakPoints
		driver.SourceCode = tt.sourceCode
		for _, bp := range tt.breakPoints {
			driver.RunUntilBreakPoint(bp.line)
			driver.StepInto()
			t.Logf(driver.State())
			t.Logf("%d", driver.VM.State())
		}

		expected := tt.expectedLocation
		vmLoc := driver.VM.SourceLocation()
		actual := vmLoc.Range.Start.Line
		if expected != actual {
			t.Errorf("error in breaktpoint test %d", i+1)
			t.Errorf("wrong breakpoint line: expected line=%d, got line=%d", expected, actual)
		}
	}
}

func TestStepOut(t *testing.T) {
	tests := []driverTestCase{
		{
			sourceCode: `
let square = fn(x) {
	return x * x
}
let squareAndDouble = fn(a) {
	let b = square(a) * 2
	return b
}
let z = square(2)
let y = squareAndDouble(2)
let bogus = 3`,

			breakPoints: []breakpoint{
				{line: 10, col: 0},
			},
			expectedLocation: 0,
		},
		{
			sourceCode: `
let square = fn(x) {
	return x * x
}
let squareAndDouble = fn(a) {
	let b = square(a) * 2
	return b
}
let z = square(2)
let y = squareAndDouble(2)
let bogus = 3`,

			breakPoints: []breakpoint{
				{line: 6, col: 0},
			},
			expectedLocation: 10,
		},
		{
			sourceCode: `
let x = 2
x`,

			breakPoints: []breakpoint{
				{line: 2, col: 0},
			},
			expectedLocation: 0,
		},
		{
			sourceCode: `
let x = 2
x`,

			breakPoints: []breakpoint{
				{line: 3, col: 0},
			},
			expectedLocation: 0,
		},
	}

	for i, tt := range tests {
		driver := New()
		err := driver.StartVM(tt.sourceCode)
		if err != nil {
			t.Errorf("error starting VM: %s", err)
		}
		driver.Breakpoints = tt.breakPoints
		driver.SourceCode = tt.sourceCode
		for _, bp := range tt.breakPoints {
			driver.RunUntilBreakPoint(bp.line)
			driver.StepOut()
			t.Logf(driver.State())
			t.Logf("%d", driver.VM.State())
		}

		expected := tt.expectedLocation
		vmLoc := driver.VM.SourceLocation()
		actual := vmLoc.Range.Start.Line
		if expected != actual {
			t.Errorf("error in breaktpoint test %d", i+1)
			t.Errorf("wrong breakpoint line: expected line=%d, got line=%d", expected, actual)
		}
	}
}

type StackFrameTestCase struct {
	sourceCode     string
	breakPoint     breakpoint
	expectedFrames []DebugFrame
}

func TestStackFrames(t *testing.T) {
	tests := []StackFrameTestCase{
		{
			sourceCode: `
let arr_all = fn(array, pred) {
    let iter = fn(arr) {
        if (arr.len() == 0) {
            return true; 
        }
        if (!pred(arr.first())) {
            return false;
        } else {
            return iter(arr.rest());
        }
    };
    iter(array);
};
let res = [2, 3, 1].arr_all(fn(x) {x < 4});
let d = 4;
`,
			breakPoint: breakpoint{
				line: 16, col: 0,
			},
			expectedFrames: []DebugFrame{
				{
					Id:   0,
					Name: "main",
					Line: 16,
				},
			},
		},
		{
			sourceCode: `
let x = 2;
x;
			`,
			breakPoint: breakpoint{
				line: 3, col: 0,
			},
			expectedFrames: []DebugFrame{
				{
					Id:   0,
					Name: "main",
					Line: 3,
				},
			},
		},
		{
			sourceCode: `
let func = fn(x) {
    let res = x + 1
    return res
}
let m = func(2);
			`,
			breakPoint: breakpoint{
				line: 4, col: 0,
			},
			expectedFrames: []DebugFrame{
				{
					Id:   0,
					Name: "main",
					Line: 6,
				},
				{
					Id:   1,
					Name: "func",
					Line: 4,
				},
			},
		},
	}

	for i, tt := range tests {
		driver := New()
		err := driver.StartVM(tt.sourceCode)
		if err != nil {
			t.Errorf("error starting VM: %s", err)
		}
		_, hit := driver.RunUntilBreakPoint(tt.breakPoint.line)
		if !hit {
			t.Errorf("error in stackframe test %d", i+1)
			t.Errorf("did not hit expected breakpoint: expected=%v, got=%v", tt.breakPoint, driver.VM.SourceLocation())
		}
		actualFrames := driver.CollectFrames()
		for j, ff := range tt.expectedFrames {
			expected := ff
			actual := actualFrames[j]
			if expected.Line != actual.Line ||
				expected.Name != actual.Name ||
				expected.Id != actual.Id {
				t.Errorf("error in stackframe test %d", i+1)
				t.Errorf("wrong stackframe line: expected frame=%v, got frame=%v", expected, actual)
			}
		}
	}

}

type ErrorTestCase struct {
	sourceCode string
	breakPoint breakpoint
	expected   error
}

func TestErrorReporting(t *testing.T) {
	tests := []ErrorTestCase{
		{
			sourceCode: `
let x = a;
let y = 2;
			`,
			breakPoint: breakpoint{
				line: 3,
				col:  1,
			},
			expected: compiler.CompilerError{},
		},
		{
			sourceCode: `
let x = 4;
let y = x();
let d = 3;
			`,
			breakPoint: breakpoint{
				line: 4,
				col:  1,
			},
			expected: vm.RunTimeError{},
		},
		{
			sourceCode: `
let x = fn(a, b) {a+b};
let y = x(3);
let d = 3;
			`,
			breakPoint: breakpoint{
				line: 4,
				col:  1,
			},
			expected: vm.RunTimeError{},
		},
		{
			sourceCode: `
let x = fn(a; b) {a + b};
let y = x(3, 2);
let d = 3;
			`,
			breakPoint: breakpoint{
				line: 4,
				col:  1,
			},
			expected: parser.ParserError{},
		},
	}

	for _, tt := range tests {
		driver := New()
		var err error
		err = driver.StartVM(tt.sourceCode)
		if err == nil {
			err, _ = driver.RunUntilBreakPoint(tt.breakPoint.line)
		}
		// Test is expected to have an error here
		if err == nil {
			t.Errorf("expected error state for code=%s, got=%s", tt.sourceCode, driver.VMState())
		}

		switch tt.expected.(type) {
		case parser.ParserError:
			actual, ok := err.(parser.ParserError)
			if !ok {
				t.Errorf("expected parser error, got=%T", actual)
			}
		case compiler.CompilerError:
			actual, ok := err.(compiler.CompilerError)
			if !ok {
				t.Errorf("expected compiler error, got=%T", actual)
			}
		case vm.RunTimeError:
			actual, ok := err.(vm.RunTimeError)
			if !ok {
				t.Errorf("expected runtime error, got=%T", actual)
			}
		}

	}
}

// TODO write a test for this code :
//let null = [1, 2][2];

//let Option = fn(x) {
//if (x == null) {
//return fn() {};
//} else {
//return fn() {x};
//}
//};

//let gett = fn(arr, i) {
//return Option(arr[i]);
//};

//let optElem = gett([1, 2], 1); STEP OVER HERE, suspect panic in vm.SourceLocation() or in Index access op code
//let optMapped = optElem.optionMap(double);
//puts(optMapped.unwrap())
//puts([1, 2].arr_any {x -> x > 1})
