package driver

import (
	"fmt"
	"monkey/compiler"
	"monkey/lexer"
	"monkey/parser"
	"monkey/vm"
)

type Driver struct {
	VM *vm.VM
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
	vm := vm.NewWithLocations(compiler.Bytecode(), compiler.LocationMap)
	d.VM = vm
	return nil
}

func (d *Driver) RunUntilBreakPoint(line int) error {
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
	return nil
}

func (d Driver) VMLocation() int {
	return d.VM.SourceLocation().Range.Start.Line
}
