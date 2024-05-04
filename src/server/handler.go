package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"monkeylang-debug/driver"

	"github.com/google/go-dap"
)

type MonkeyHandler struct {
	session             *Session
	Driver              *driver.Driver
	bpSetMux            sync.Mutex
	terminateOnNextStep bool
}

func NewHandler() MonkeyHandler {
	return MonkeyHandler{
		Driver: driver.New(),
	}
}

func (h *MonkeyHandler) SetSession(s *Session) {
	h.session = s
}

func (h *MonkeyHandler) OnInitializeRequest(request *dap.InitializeRequest) {
	response := &dap.InitializeResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body.SupportsConfigurationDoneRequest = true
	response.Body.SupportsFunctionBreakpoints = false
	response.Body.SupportsConditionalBreakpoints = false
	response.Body.SupportsHitConditionalBreakpoints = false
	response.Body.SupportsEvaluateForHovers = false
	response.Body.ExceptionBreakpointFilters = []dap.ExceptionBreakpointsFilter{}
	response.Body.SupportsStepBack = false
	response.Body.SupportsSetVariable = false
	response.Body.SupportsRestartFrame = false
	response.Body.SupportsGotoTargetsRequest = false
	response.Body.SupportsStepInTargetsRequest = false
	response.Body.SupportsCompletionsRequest = false
	response.Body.CompletionTriggerCharacters = []string{}
	response.Body.SupportsModulesRequest = false
	response.Body.AdditionalModuleColumns = []dap.ColumnDescriptor{}
	response.Body.SupportedChecksumAlgorithms = []dap.ChecksumAlgorithm{}
	response.Body.SupportsRestartRequest = false
	response.Body.SupportsExceptionOptions = false
	response.Body.SupportsValueFormattingOptions = false
	response.Body.SupportsExceptionInfoRequest = true
	response.Body.SupportTerminateDebuggee = false
	response.Body.SupportsDelayedStackTraceLoading = false
	response.Body.SupportsLoadedSourcesRequest = false
	response.Body.SupportsLogPoints = false
	response.Body.SupportsTerminateThreadsRequest = false
	response.Body.SupportsSetExpression = false
	response.Body.SupportsTerminateRequest = false
	response.Body.SupportsDataBreakpoints = false
	response.Body.SupportsReadMemoryRequest = false
	response.Body.SupportsDisassembleRequest = false
	response.Body.SupportsCancelRequest = false
	response.Body.SupportsBreakpointLocationsRequest = false
	// This is a fake set up, so we can start "accepting" configuration
	// requests for setting breakpoints, etc from the client at any time.
	// Notify the client with an 'initialized' event. The client will end
	// the configuration sequence with 'configurationDone' request.
	e := &dap.InitializedEvent{Event: *newEvent("initialized")}
	h.session.send(e)
	h.session.send(response)
}

func (h *MonkeyHandler) OnLaunchRequest(request *dap.LaunchRequest) {
	code, err := os.ReadFile(h.session.source.Path)
	if err != nil {
		panic(fmt.Sprintf("could not read source file=%s", h.session.source.Path))
	}

	err = h.Driver.StartVM(string(code))
	if err != nil {
		response := &dap.LaunchResponse{}
		response.Response = *newResponse(request.Seq, request.Command)
		h.session.send(response)
		log.Printf("could not start vm: %s", err)

		go func() {
			time.Sleep(200 * time.Millisecond)
			s := h.Driver.State()
			log.Printf("State: %d", s)
			switch s {
			case driver.OFF:
			default:
				e := h.ProduceStopEvent(h.Driver.State())
				h.session.send(e)
			}
		}()
		return
	}
	log.Printf("started vm with code=%s\n", string(code))

	go func() {
		time.Sleep(200 * time.Millisecond)
		err, hit := h.Driver.RunWithBreakpoints(h.Driver.Breakpoints)
		if err != nil {
			log.Printf("error runnig VM: %s", err)
		}
		log.Printf("bp hit=%v", hit)
		log.Printf("Ran VM until %v\n", h.Driver.VM.SourceLocation())
		log.Printf("VM state=%s", h.Driver.State().String())
		log.Printf("VM pointer=%d", h.Driver.VM.CurrentFrame().Ip)

		switch h.Driver.State() {
		case driver.OFF:
			return
		default:
			e := h.ProduceStopEvent(h.Driver.State())
			h.session.send(e)
		}
	}()

	response := &dap.LaunchResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
}

func (h *MonkeyHandler) OnAttachRequest(request *dap.AttachRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "AttachRequest is not yet supported"))
}

func (h *MonkeyHandler) OnDisconnectRequest(request *dap.DisconnectRequest) {
	response := &dap.DisconnectResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.stopDebug <- struct{}{}
	h.session.send(response)
}

func (h *MonkeyHandler) OnTerminateRequest(request *dap.TerminateRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "TerminateRequest is not yet supported"))
}

func (h *MonkeyHandler) OnRestartRequest(request *dap.RestartRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "RestartRequest is not yet supported"))
}

func (h *MonkeyHandler) OnSetBreakpointsRequest(request *dap.SetBreakpointsRequest) {
	bps := request.Arguments.Breakpoints
	lines := make([]int, len(bps))
	for i, bp := range bps {
		lines[i] = bp.Line
	}
	h.Driver.SetBreakPoints(lines)

	source := request.Arguments.Source
	h.Driver.Source = source.Path
	h.session.source = source

	response := &dap.SetBreakpointsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body.Breakpoints = make([]dap.Breakpoint, len(request.Arguments.Breakpoints))
	for i, b := range request.Arguments.Breakpoints {
		response.Body.Breakpoints[i].Line = b.Line
		response.Body.Breakpoints[i].Verified = true
	}
	h.session.send(response)
}

func (h *MonkeyHandler) OnSetFunctionBreakpointsRequest(request *dap.SetFunctionBreakpointsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SetFunctionBreakpointsRequest is not yet supported"))
}

func (h *MonkeyHandler) OnSetExceptionBreakpointsRequest(request *dap.SetExceptionBreakpointsRequest) {
	response := &dap.SetExceptionBreakpointsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
}

func (h *MonkeyHandler) OnConfigurationDoneRequest(request *dap.ConfigurationDoneRequest) {
	e := &dap.ThreadEvent{Event: *newEvent("thread"), Body: dap.ThreadEventBody{Reason: "started", ThreadId: 1}}
	h.session.send(e)
	response := &dap.ConfigurationDoneResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
}

func (h *MonkeyHandler) OnContinueRequest(request *dap.ContinueRequest) {
	response := &dap.ContinueResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)

	h.bpSetMux.Lock()

	bps := h.Driver.Breakpoints
	log.Printf("Breakpoints: %v", bps)
	err, _ := h.Driver.RunWithBreakpoints(bps)
	if err != nil {
		log.Printf("error runnig VM: %s", err)
	}
	log.Printf("Ran VM until %v\n", h.Driver.VM.SourceLocation())

	s := h.Driver.State()
	switch s {
	case driver.OFF:
		log.Printf("%d\n", s)
		return
	default:
		log.Printf("%d\n", s)
		e := h.ProduceStopEvent(s)
		h.session.send(e)
	}

	h.bpSetMux.Unlock()
}

func (h *MonkeyHandler) OnNextRequest(request *dap.NextRequest) {
	acknowledgement := &dap.NextResponse{}
	acknowledgement.Response = *newResponse(request.Seq, request.Command)
	h.session.send(acknowledgement)

	log.Printf("sent acknowledgement")

	command := request.Command
	log.Printf("Received command=%s", command)
	err, _ := h.Driver.StepOver()
	log.Printf("Ran VM until %v\n", h.Driver.VM.SourceLocation())
	log.Printf("VM State=%s", h.Driver.State().String())

	if err != nil {
		log.Printf("error handling NextRequest: %s", err)
	}

	switch h.Driver.State() {
	case driver.OFF:
		return
	default:
		e := h.ProduceStopEvent(h.Driver.State())
		h.session.send(e)
	}

}

func (h *MonkeyHandler) OnStepInRequest(request *dap.StepInRequest) {
	acknowledgement := &dap.StepInResponse{}
	acknowledgement.Response = *newResponse(request.Seq, request.Command)
	h.session.send(acknowledgement)

	log.Printf("sent acknowledgement")

	command := request.Command
	log.Printf("Received command=%s", command)
	err, _ := h.Driver.StepInto()
	log.Printf("Ran VM until %v\n", h.Driver.VM.SourceLocation())
	log.Printf("VM State=%s", h.Driver.State().String())

	if err != nil {
		log.Printf("error handling NextRequest: %s", err)
	}

	switch h.Driver.State() {
	case driver.OFF:
		return
	default:
		e := h.ProduceStopEvent(h.Driver.State())
		h.session.send(e)
	}

}

func (h *MonkeyHandler) OnStepOutRequest(request *dap.StepOutRequest) {
	acknowledgement := &dap.StepOutResponse{}
	acknowledgement.Response = *newResponse(request.Seq, request.Command)
	h.session.send(acknowledgement)

	log.Printf("sent acknowledgement")

	command := request.Command
	log.Printf("Received command=%s", command)
	err, _ := h.Driver.StepOut()
	log.Printf("Ran VM until %v\n", h.Driver.VM.SourceLocation())
	log.Printf("VM State=%s", h.Driver.State().String())

	if err != nil {
		log.Printf("error handling NextRequest: %s", err)
	}

	switch h.Driver.State() {
	case driver.OFF:
		return
	default:
		e := h.ProduceStopEvent(h.Driver.State())
		h.session.send(e)
	}
}

func (h *MonkeyHandler) OnStepBackRequest(request *dap.StepBackRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "StepBackRequest is not yet supported"))
}

func (h *MonkeyHandler) OnReverseContinueRequest(request *dap.ReverseContinueRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "ReverseContinueRequest is not yet supported"))
}

func (h *MonkeyHandler) OnRestartFrameRequest(request *dap.RestartFrameRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "RestartFrameRequest is not yet supported"))
}

func (h *MonkeyHandler) OnGotoRequest(request *dap.GotoRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "GotoRequest is not yet supported"))
}

func (h *MonkeyHandler) OnPauseRequest(request *dap.PauseRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "PauseRequest is not yet supported"))
}

func (h *MonkeyHandler) OnStackTraceRequest(request *dap.StackTraceRequest) {
	response := &dap.StackTraceResponse{}
	response.Response = *newResponse(request.Seq, request.Command)

	ds := h.Driver.State()
	log.Printf("Driver state: %v", ds)
	switch ds {
	case driver.COMPILER_ERROR:
		e := h.Driver.Errors[0]
		stackFrames := make([]dap.StackFrame, 1)
		stackFrames[0] = dap.StackFrame{
			Id:     0,
			Name:   "Compiler Error",
			Source: &h.session.source,
			Line:   e.Line(),
			Column: e.Col(),
		}
		response.Body = dap.StackTraceResponseBody{
			StackFrames: stackFrames,
			TotalFrames: 1,
		}
	default:
		driverFrames := h.Driver.CollectFrames()
		stackFrames := make([]dap.StackFrame, len(driverFrames))
		// reverse the order: deepest stack frame must be first in array
		for i := len(stackFrames) - 1; i >= 0; i-- {
			stackFrames[i] = h.DriverFrameToStackFrame(driverFrames[len(stackFrames)-1-i])
		}
		response.Body = dap.StackTraceResponseBody{
			StackFrames: stackFrames,
			TotalFrames: len(stackFrames),
		}
	}

	h.session.send(response)
}

func (h *MonkeyHandler) OnScopesRequest(request *dap.ScopesRequest) {

	frameId := request.Arguments.FrameId

	response := &dap.ScopesResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	scopes := []dap.Scope{}
	if frameId > 0 {
		localScope := dap.Scope{Name: "Local", VariablesReference: frameId + 1, Expensive: false}
		scopes = append(scopes, localScope)
	}
	//always attach global scope
	scopes = append(scopes, dap.Scope{Name: "Global", VariablesReference: 1, Expensive: false})

	response.Body = dap.ScopesResponseBody{
		Scopes: scopes,
	}
	select {
	case <-h.session.stopDebug:
		return
	default:
		h.session.send(response)
	}
}

func (h *MonkeyHandler) OnVariablesRequest(request *dap.VariablesRequest) {
	// subtract 1 from ref and use the value as an index into our driver frames
	varRef := request.Arguments.VariablesReference - 1

	driverVars := h.Driver.Frames[varRef].Vars
	log.Printf("driverVars: %v", driverVars)
	vars := make([]dap.Variable, len(driverVars))
	for i, dv := range driverVars {
		vars[i] = DriverVarToDAPVar(dv)
	}
	select {
	case <-h.session.stopDebug:
		return
	default:
		response := &dap.VariablesResponse{}
		response.Response = *newResponse(request.Seq, request.Command)
		response.Body = dap.VariablesResponseBody{
			Variables: vars,
		}
		h.session.send(response)
	}
}

func (h *MonkeyHandler) OnSetVariableRequest(request *dap.SetVariableRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "setVariableRequest is not yet supported"))
}

func (h *MonkeyHandler) OnSetExpressionRequest(request *dap.SetExpressionRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SetExpressionRequest is not yet supported"))
}

func (h *MonkeyHandler) OnSourceRequest(request *dap.SourceRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SourceRequest is not yet supported"))
}

func (h *MonkeyHandler) OnThreadsRequest(request *dap.ThreadsRequest) {
	response := &dap.ThreadsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body = dap.ThreadsResponseBody{Threads: []dap.Thread{{Id: 1, Name: "main"}}}
	h.session.send(response)

}

func (h *MonkeyHandler) OnTerminateThreadsRequest(request *dap.TerminateThreadsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "TerminateRequest is not yet supported"))
}

func (h *MonkeyHandler) OnEvaluateRequest(request *dap.EvaluateRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "EvaluateRequest is not yet supported"))
}

func (h *MonkeyHandler) OnStepInTargetsRequest(request *dap.StepInTargetsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "StepInTargetRequest is not yet supported"))
}

func (h *MonkeyHandler) OnGotoTargetsRequest(request *dap.GotoTargetsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "GotoTargetRequest is not yet supported"))
}

func (h *MonkeyHandler) OnCompletionsRequest(request *dap.CompletionsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "CompletionRequest is not yet supported"))
}

func (h *MonkeyHandler) OnExceptionInfoRequest(request *dap.ExceptionInfoRequest) {
	response := &dap.ExceptionInfoResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	body := dap.ExceptionInfoResponseBody{}
	body.BreakMode = "always"
	body.Description = h.Driver.Errors[0].Error()
	body.Details = &dap.ExceptionDetails{
		Message: h.Driver.Errors[0].Error(),
	}
	response.Body = body
	h.session.send(response)
}

func (h *MonkeyHandler) OnLoadedSourcesRequest(request *dap.LoadedSourcesRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "LoadedRequest is not yet supported"))
}

func (h *MonkeyHandler) OnDataBreakpointInfoRequest(request *dap.DataBreakpointInfoRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "DataBreakpointInfoRequest is not yet supported"))
}

func (h *MonkeyHandler) OnSetDataBreakpointsRequest(request *dap.SetDataBreakpointsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SetDataBreakpointsRequest is not yet supported"))
}

func (h *MonkeyHandler) OnReadMemoryRequest(request *dap.ReadMemoryRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "ReadMemoryRequest is not yet supported"))
}

func (h *MonkeyHandler) OnDisassembleRequest(request *dap.DisassembleRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "DisassembleRequest is not yet supported"))
}

func (h *MonkeyHandler) OnCancelRequest(request *dap.CancelRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "CancelRequest is not yet supported"))
}

func (h *MonkeyHandler) OnBreakpointLocationsRequest(request *dap.BreakpointLocationsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "BreakpointLocationsRequest is not yet supported"))
}

func DriverVarToDAPVar(driverVar driver.DriverVar) dap.Variable {
	return dap.Variable{
		Name:               driverVar.Name,
		Value:              driverVar.Value,
		VariablesReference: driverVar.VariablesReference,
		Type:               driverVar.Type,
	}
}

func (h *MonkeyHandler) DriverFrameToStackFrame(driverFrame driver.DebugFrame) dap.StackFrame {
	return dap.StackFrame{
		Id:     driverFrame.Id,
		Name:   driverFrame.Name,
		Source: &h.session.source,
		Line:   driverFrame.Line,
		Column: driverFrame.Column,
	}

}

func (h *MonkeyHandler) ProduceStopEvent(state driver.State) dap.Message {
	// switch on the State here
	log.Printf("producing stop event for state=%s", state.String())
	var e dap.Message

	if h.terminateOnNextStep {
		e = &dap.TerminatedEvent{
			Event: *newEvent("terminated"),
		}
		return e
	}

	st := h.Driver.State()
	switch st {
	case driver.STOPPED:
		e = &dap.StoppedEvent{
			Event: *newEvent("stopped"),
			Body: dap.StoppedEventBody{
				Reason:   "breakpoint",
				ThreadId: 1, AllThreadsStopped: true,
			},
		}
	case driver.COMPILER_ERROR, driver.RUNTIME_ERROR:
		e = &dap.StoppedEvent{
			Event: *newEvent("stopped"),
			Body: dap.StoppedEventBody{
				Reason:            "exception",
				ThreadId:          1,
				AllThreadsStopped: true,
				Description:       st.String(),
			},
		}
		h.terminateOnNextStep = true
	case driver.DONE:
		e = &dap.TerminatedEvent{
			Event: *newEvent("terminated"),
		}
	}
	return e
}
