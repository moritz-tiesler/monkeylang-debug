package driver

import (
	"testing"
)

type driverTestCase struct {
	sourceCode  string
	breakPoints []breakpoint
}

func TestBreakPoints(t *testing.T) {
	tests := []driverTestCase{
		{
			sourceCode: `let square = fn(x) {
	return x * x
}
let squareAndDouble = fn(a) {
	let b = square(a) * 2
	return b
}
squareAndDouble(2)
square(2)`,

			breakPoints: []breakpoint{
				{line: 2, col: 0},
				{line: 8, col: 0},
			},
		},
	}

	for i, tt := range tests {
		driver := New()
		err := driver.StartVM(tt.sourceCode)
		if err != nil {
			t.Errorf("error starting VM: %s", err)
		}
		for _, bp := range tt.breakPoints {
			driver.RunUntilBreakPoint(bp.line)
		}

		expected := tt.breakPoints[len(tt.breakPoints)-1].line
		actual := driver.VM.SourceLocation().Range.Start.Line + 1
		if expected != actual {
			t.Errorf("error in breaktpoint test %d", i)
			t.Errorf("wrong breakpoint line: expected=%d, got=%d", expected, actual)
		}
	}
}
