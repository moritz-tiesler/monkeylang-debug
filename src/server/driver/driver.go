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
		if vmLoc.Range.Start.Line  == lineNum {
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

//cfunc (d *Driver) saveBreakpoint(line int) {
//cbp := breakpoint{line: line, col: 0}
//cexistingBreakpoints := d.Breakpoints[d.VM.CurrentFrame().Closure().Fn]
//cd.Breakpoints[d.VM.CurrentFrame().Closure().Fn] = append(existingBreakpoints, bp)
//c}

//func (d *Driver) runSavedBreakpoints() error {
	//for _, bps := range d.Breakpoints {
		//for _, bp := range bps {
			//line := bp.line

			//runCondition := func(vm *vm.VM) (bool, error) {
				//executionLine := vm.SourceLocation().Range.Start.Line
				//if line == executionLine {
					//vm.CurrentFrame().Ip--
					//return true, nil
				//} else {
					//return false, nil
				//}
			//}

			//vm, err := d.VM.RunWithCondition(runCondition)
			//if err != nil {
				//return err
			//}
			//d.VM = vm
		//}
	//}
	//return nil
//}

func (d *Driver) StepOver() error  {
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

func (d *Driver) RunWithBreakpoints(bps []breakpoint) error  {
	//runCondition := func(vm *vm.VM) (bool, error) {
	//executionLine := vm.SourceLocation().Range.Start.Line
	//for _, bp := range bps {
	//if executionLine == bp.line-1 {
	//vm.CurrentFrame().Ip--
	//return true, nil
	//}

	//}
	//return false, nil
	//}

	// TODO: check whether the vm is currently at a breakpoint and if so, cycle once to
	// avoid hitting the same breakpoint again immideatly
	if d.stoppedOnBreakpoint {
		d.StepOver()
	}
	
	d.Breakpoints = bps
	for _, bp := range bps {
		tempCopy := d.VM.Copy()
		err, hitBp := d.RunUntilBreakPoint(bp.line)
		if err != nil {
			return err 
		}
		if hitBp{
			d.stoppedOnBreakpoint = true
			break
		// TODO: if bp was not hit run the next bp with a copy of the vm BEFORE the previous bp was tried
		} else {
			d.VM = tempCopy 
			d.stoppedOnBreakpoint = false
			continue
		}
	}

	return nil 
}

func (d *Driver) RunUntilBreakPoint(line int) (error, bool) {
	//err := d.runSavedBreakpoints()
	//if err != nil {
	//return err
	//}
	runCondition := func(vm *vm.VM) (bool, error) {
		executionLoc := vm.SourceLocation()
		executionLine := executionLoc.Range.Start.Line
		if line == executionLine {
			//d.saveBreakpoint(executionLine)
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

func (d Driver) VMLocation() int{

	loc := d.VM.SourceLocation()
	return loc.Range.End.Line 
}
