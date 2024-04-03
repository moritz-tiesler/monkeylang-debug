package driver

import (
	"fmt"
	"monkey/compiler"
	"monkey/lexer"
	"monkey/parser"
	"monkey/vm"
	"strings"
)

type breakpoint struct {
	line int
	col  int
	doc  string
}

type Driver struct {
	VM                  *vm.VM
	Breakpoints         []breakpoint
	Source              string
	SourceCode          string
	stoppedOnBreakpoint bool
}

type vmState int

const (
	OFF vmState = iota
	STOPPED
	DONE
)

func (d Driver) VMState() vmState {

	return vmState(d.VM.State())
}

func (st vmState) String() string {
	var s string
	switch st {
	case OFF:
		s = "OFF"
	case STOPPED:
		s = "STOPPED"
	case DONE:
		s = "DONE"
	}

	return s
}

func New() *Driver {
	return &Driver{
		Breakpoints:         make([]breakpoint, 0),
		stoppedOnBreakpoint: false,
	}
}

func (d *Driver) SetBreakPoints(lines []int) {
	bps := make([]breakpoint, len(lines))
	for i, l := range lines {
		bps[i] = breakpoint{line: l}
	}
	d.Breakpoints = bps
}

func (d *Driver) State() string {
	state := ""

	lines := strings.Split(d.SourceCode, "\n")
	for i, l := range lines {
		padding := ""
		lineNum := i + 1
		for _, bp := range d.Breakpoints {
			if bp.line == lineNum {
				padding = padding + "#"
				break
			}
		}
		vmLoc := d.VM.SourceLocation()
		if vmLoc.Range.Start.Line == lineNum {
			padding = padding + "->"
		}

		for len(padding) < 4 {
			padding = padding + " "
		}

		state = state + "\n" + padding + l

	}
	return state
}

func (d *Driver) StartVM(sourceCode string) error {
	lexer := lexer.New(sourceCode)
	parser := parser.New(lexer)
	program := parser.ParseProgram()
	compiler := compiler.New()
	err := compiler.Compile(program)
	if err != nil {
		return fmt.Errorf("compilation error")
	}
	vm := vm.NewFromMain(compiler.MainFn(), compiler.Bytecode(), compiler.LocationMap)
	d.VM = vm
	return nil
}

func (d *Driver) StepOver() (error, bool) {
	staringLoc := d.VM.SourceLocation()
	startingLine := staringLoc.Range.Start.Line
	startingDepth := d.VM.CallDepth

	runCondition := func(vm *vm.VM) (bool, error) {
		cycleLocation := vm.SourceLocation()
		cycleLine := cycleLocation.Range.Start.Line
		cycleDepth := d.VM.CallDepth
		if cycleLine != startingLine && cycleDepth <= startingDepth {
			if !(d.VMState() == DONE) {
				vm.CurrentFrame().Ip--
			}
			return true, nil
		} else {
			return false, nil
		}
	}

	vm, err, conditonMet := d.VM.RunWithCondition(runCondition)
	if err != nil {
		return err, false
	}
	d.VM = vm
	d.stoppedOnBreakpoint = false
	return nil, conditonMet
}

func (d *Driver) StepInto() (error, bool) {
	staringLoc := d.VM.SourceLocation()
	startingLine := staringLoc.Range.Start.Line
	startingDepth := d.VM.CallDepth

	runCondition := func(vm *vm.VM) (bool, error) {
		cycleLocation := vm.SourceLocation()
		cycleLine := cycleLocation.Range.Start.Line
		cycleDepth := d.VM.CallDepth

		// If we call StepInto on a line that has nothing to step into
		// essentially the same as stepping over
		if (cycleLine != startingLine && cycleDepth <= startingDepth) ||
			// If we call StepInto on a line that has smth to step into
			(cycleDepth > startingDepth) {
			if !(d.VMState() == DONE) {
				vm.CurrentFrame().Ip--
			}
			return true, nil
		} else {
			return false, nil
		}
	}

	vm, err, conditonMet := d.VM.RunWithCondition(runCondition)
	if err != nil {
		return err, false
	}
	d.VM = vm
	d.stoppedOnBreakpoint = false
	return nil, conditonMet
}

func (d *Driver) StepOut() (error, bool) {
	//staringLoc := d.VM.SourceLocation()
	//startingLine := staringLoc.Range.Start.Line
	startingDepth := d.VM.CallDepth

	runCondition := func(vm *vm.VM) (bool, error) {
		//cycleLocation := vm.SourceLocation()
		//cycleLine := cycleLocation.Range.Start.Line
		cycleDepth := d.VM.CallDepth

		if cycleDepth < startingDepth {
			if !(d.VMState() == DONE) {
				vm.CurrentFrame().Ip--
			}
			return true, nil
		} else {
			return false, nil
		}
	}

	vm, err, conditonMet := d.VM.RunWithCondition(runCondition)
	if err != nil {
		return err, false
	}
	d.VM = vm
	d.stoppedOnBreakpoint = false
	return nil, conditonMet
}
func (d *Driver) RunWithBreakpoints(bps []breakpoint) (error, bool) {

	if d.stoppedOnBreakpoint {
		d.StepOver()
	}

	runCondition := func(vm *vm.VM) (bool, error) {
		executionLoc := vm.SourceLocation()
		executionLine := executionLoc.Range.Start.Line
		for _, bp := range bps {

			if bp.line == executionLine {
				d.stoppedOnBreakpoint = true
				vm.CurrentFrame().Ip--
				return true, nil
			}
		}
		d.stoppedOnBreakpoint = false
		return false, nil
	}

	vm, err, breakPointHit := d.VM.RunWithCondition(runCondition)
	if err != nil {
		return err, false
	}
	d.VM = vm

	return nil, breakPointHit
}

func (d *Driver) RunUntilBreakPoint(line int) (error, bool) {
	runCondition := func(vm *vm.VM) (bool, error) {
		executionLoc := vm.SourceLocation()
		executionLine := executionLoc.Range.Start.Line
		if line == executionLine {
			vm.CurrentFrame().Ip--
			return true, nil
		} else {
			return false, nil
		}
	}

	vm, err, breakPointHit := d.VM.RunWithCondition(runCondition)
	if err != nil {
		return err, false
	}

	d.VM = vm
	return nil, breakPointHit
}

func (d Driver) VMLocation() int {

	loc := d.VM.SourceLocation()
	return loc.Range.End.Line
}

type DebugFrame struct {
	Id     int
	Name   string
	Source string
	Line   int
	Column int
}

func (d Driver) NewDebugFrame(id int, vmFrame *vm.Frame) DebugFrame {
	name := vmFrame.Name()
	source := d.Source
	key := compiler.LocationKey{ScopeId: vmFrame.Closure().Fn, InstructionIndex: vmFrame.Ip + 1}
	loc := d.VM.LocationMap[key]
	line := loc.Range.Start.Line
	col := loc.Range.Start.Col

	return DebugFrame{
		Id:     id,
		Name:   name,
		Source: source,
		Line:   line,
		Column: col,
	}
}

func (d Driver) CollectFrames() []DebugFrame {
	numFrames := d.VM.FramesIndex()
	frames := make([]DebugFrame, numFrames)
	for i := 0; i < numFrames; i++ {
		vmFrame := d.VM.Frames()[i]
		debugFrame := d.NewDebugFrame(i, vmFrame)
		if i == 0 {
			debugFrame.Name = "main"
		}
		debugFrame.Source = d.Source
		frames[i] = debugFrame
	}
	return frames
}
