package driver

import (
	"fmt"
	"monkey/compiler"
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
	"monkey/vm"
)

type breakpoint struct {
	line int
	col  int
	doc  string
}

type Driver struct {
	VM          *vm.VM
	Breakpoints map[*object.CompiledFunction][]breakpoint
	Source      string
}

func New() *Driver {
	return &Driver{
		Breakpoints: make(map[*object.CompiledFunction][]breakpoint),
	}
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

func (d *Driver) saveBreakpoint(line int) {
	bp := breakpoint{line: line, col: 0}
	existingBreakpoints := d.Breakpoints[d.VM.CurrentFrame().Closure().Fn]
	d.Breakpoints[d.VM.CurrentFrame().Closure().Fn] = append(existingBreakpoints, bp)
}

func (d *Driver) runSavedBreakpoints() error {
	for _, bps := range d.Breakpoints {
		for _, bp := range bps {
			line := bp.line

			runCondition := func(vm *vm.VM) (bool, error) {
				executionLine := vm.SourceLocation().Range.Start.Line
				if line == executionLine {
					vm.CurrentFrame().Ip--
					return true, nil
				} else {
					return false, nil
				}
			}

			vm, err := d.VM.RunWithCondition(runCondition)
			if err != nil {
				return err
			}
			d.VM = vm
		}
	}
	return nil
}

func (d *Driver) RunUntilBreakPoint(line int) error {
	err := d.runSavedBreakpoints()
	if err != nil {
		return err
	}
	line = line - 1
	runCondition := func(vm *vm.VM) (bool, error) {
		executionLine := vm.SourceLocation().Range.Start.Line
		if line == executionLine {
			d.saveBreakpoint(executionLine)
			vm.CurrentFrame().Ip--
			return true, nil
		} else {
			return false, nil
		}
	}

	vm, err := d.VM.RunWithCondition(runCondition)
	if err != nil {
		return err
	}

	d.VM = vm
	return nil
}

func (d Driver) VMLocation() int {
	return d.VM.SourceLocation().Range.Start.Line + 1
}
