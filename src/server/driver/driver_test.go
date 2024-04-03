package driver

import (
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
let y = squareAndDouble(2)`,

			breakPoints: []breakpoint{
				{line: 2, col: 0},
				{line: 9, col: 0},
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
		driver.RunUntilBreakPoint(tt.breakPoint.line)
		t.Logf(driver.State())
		t.Logf("%d", driver.VM.State())

		for j, ff := range tt.expectedFrames {
			expected := ff
			actual := driver.CollectFrames()[j]
			if expected.Line != actual.Line ||
				expected.Name != actual.Name ||
				expected.Id != actual.Id {
				t.Errorf("error in stackframe test %d", i+1)
				t.Errorf("wrong stackframe line: expected frame=%v, got frame=%v", expected, actual)
			}
		}
	}

}
