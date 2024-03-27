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

func New() *Driver {
	return &Driver{
		Breakpoints:         make([]breakpoint, 0),
		stoppedOnBreakpoint: false,
	}
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

func (d *Driver) StepOver() error {
	staringLoc := d.VM.SourceLocation()
	startingLine := staringLoc.Range.Start.Line
	runCondition := func(vm *vm.VM) (bool, error) {
		cycleLocation := vm.SourceLocation()
		cycleLine := cycleLocation.Range.Start.Line
		if cycleLine != startingLine {
			vm.CurrentFrame().Ip--
			return true, nil
		} else {
			return false, nil
		}
	}

	vm, err, _ := d.VM.RunWithCondition(runCondition)
	if err != nil {
		return err
	}
	d.VM = vm
	d.stoppedOnBreakpoint = false
	return nil
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
				//d.saveBreakpoint(executionLine)
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
