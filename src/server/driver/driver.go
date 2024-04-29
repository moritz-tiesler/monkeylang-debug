package driver

import (
	"monkey/compiler"
	"monkey/exception"
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
	"monkey/vm"
	"strings"
)

type breakpoint struct {
	line int
	col  int
}

type Driver struct {
	VM                  *vm.VM
	Breakpoints         []breakpoint
	Source              string
	SourceCode          string
	stoppedOnBreakpoint bool
	Frames              []DebugFrame
	Errors              []exception.Exception
}

type VMState int

const (
	OFF VMState = iota
	STOPPED
	DONE
	ERROR
)

func (d Driver) VMState() VMState {

	if d.HasErrors() {
		return ERROR
	}
	return VMState(d.VM.State())
}

func (st VMState) String() string {
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

func (d Driver) HasErrors() bool {
	return len(d.Errors) > 0
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
	parserErrors := parser.Errors()
	if len(parserErrors) > 0 {
		d.Errors = append(d.Errors, parserErrors...)
		return d.Errors[0]
	}
	compiler := compiler.New()
	err := compiler.Compile(program)
	if err != nil {
		d.Errors = append(d.Errors, err)
		return err
	}
	vm := vm.NewFromMain(compiler.MainFn(), compiler.Bytecode(), compiler.LocationMap, compiler.NameStore)
	d.VM = vm
	return nil
}

func (d *Driver) StepOver() (error, bool) {
	staringLoc := d.VM.SourceLocation()
	startingLine := staringLoc.Range.Start.Line
	startingDepth := d.VM.CallDepth

	runCondition := func(vm *vm.VM) (bool, exception.Exception) {
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
		d.Errors = append(d.Errors, err)
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

	runCondition := func(vm *vm.VM) (bool, exception.Exception) {
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
		d.Errors = append(d.Errors, err)
		return err, false
	}
	d.VM = vm
	d.stoppedOnBreakpoint = false
	return nil, conditonMet
}

func (d *Driver) StepOut() (error, bool) {
	startingDepth := d.VM.CallDepth

	runCondition := func(vm *vm.VM) (bool, exception.Exception) {
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
		d.Errors = append(d.Errors, err)
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

	runCondition := func(vm *vm.VM) (bool, exception.Exception) {
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
		d.Errors = append(d.Errors, err)
		return err, false
	}
	d.VM = vm

	return nil, breakPointHit
}

func (d *Driver) RunUntilBreakPoint(line int) (error, bool) {
	runCondition := func(vm *vm.VM) (bool, exception.Exception) {
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
		d.Errors = append(d.Errors, err)
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
	Vars   []DriverVar
}

func (d Driver) NewDebugFrame(id int, vmFrame *vm.Frame) DebugFrame {
	name := vmFrame.Name()
	source := d.Source
	loc := d.VM.SourceLocationInFrame(vmFrame)
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

func (d *Driver) CollectFrames() []DebugFrame {
	numFrames := d.VM.FramesIndex()
	vmFrames := d.VM.Frames()
	debugFrames := make([]DebugFrame, numFrames)
	for i := 0; i < numFrames; i++ {
		vmFrame := vmFrames[i]
		debugFrame := d.NewDebugFrame(i, vmFrame)
		if i == 0 {
			debugFrame.Name = "main"
		}
		debugFrame.Source = d.Source

		frameObjects, names := d.VM.ActiveObjects(*vmFrame)

		frameVars := make([]DriverVar, len(frameObjects))
		for j, obj := range frameObjects {
			name := names[obj]
			frameVars[j] = ObjectToDriverVar(obj, name)
		}
		debugFrame.Vars = frameVars
		debugFrames[i] = debugFrame

	}
	d.Frames = debugFrames
	return debugFrames
}

type DriverVar struct {
	Name               string
	Value              string
	Type               string
	VariablesReference int
}

func ObjectToDriverVar(obj object.Object, name string) DriverVar {
	v := DriverVar{
		Name:               name,
		VariablesReference: 0,
	}
	switch obj := obj.(type) {
	case *object.Closure:
		v.Value = "function"
		v.Type = "function"
	default:
		v.Value = obj.Inspect()
		v.Type = string(obj.Type())
	}
	return v
}
