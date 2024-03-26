package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"monkeylang-debug/driver"

	"github.com/google/go-dap"
)

type MonkeyHandler struct {
	session *Session
	Driver  *driver.Driver
}

func NewHandler() MonkeyHandler {
	return MonkeyHandler{
		Driver: driver.New(),
	}
}

func (h *MonkeyHandler) SetSession(s *Session) {
	h.session = s
}

func (h MonkeyHandler) OnInitializeRequest(request *dap.InitializeRequest) {
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
	response.Body.SupportsExceptionInfoRequest = false
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

func (h MonkeyHandler) OnLaunchRequest(request *dap.LaunchRequest) {
	// This is where a real debug adaptor would check the soundness of the
	// arguments (e.g. program from launch.json) and then use them to launch the
	// debugger and attach to the program.
	response := &dap.LaunchResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
}

func (h MonkeyHandler) OnAttachRequest(request *dap.AttachRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "AttachRequest is not yet supported"))
}

func (h MonkeyHandler) OnDisconnectRequest(request *dap.DisconnectRequest) {
	response := &dap.DisconnectResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
}

func (h MonkeyHandler) OnTerminateRequest(request *dap.TerminateRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "TerminateRequest is not yet supported"))
}

func (h MonkeyHandler) OnRestartRequest(request *dap.RestartRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "RestartRequest is not yet supported"))
}

func (h MonkeyHandler) OnSetBreakpointsRequest(request *dap.SetBreakpointsRequest) {
	source := request.Arguments.Source
	h.Driver.Source = source.Path
	h.session.source = source
	code, err := os.ReadFile(source.Path)
	if err != nil {
		panic(fmt.Sprintf("could not read source file=%s", source.Path))
	}

	err = h.Driver.StartVM(string(code))
	if err != nil {
		log.Fatalf("could not start vm")
	}
	log.Printf("running vm with code=%s\n", string(code))

	h.session.breakPoints = request.Arguments.Breakpoints
	response := &dap.SetBreakpointsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body.Breakpoints = make([]dap.Breakpoint, len(request.Arguments.Breakpoints))
	for i, b := range request.Arguments.Breakpoints {
		response.Body.Breakpoints[i].Line = b.Line
		response.Body.Breakpoints[i].Verified = true
		h.session.bpSetMux.Lock()
		h.session.bpSet++
		h.session.bpSetMux.Unlock()
	}
	h.session.send(response)
}

func (h MonkeyHandler) OnSetFunctionBreakpointsRequest(request *dap.SetFunctionBreakpointsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SetFunctionBreakpointsRequest is not yet supported"))
}

func (h MonkeyHandler) OnSetExceptionBreakpointsRequest(request *dap.SetExceptionBreakpointsRequest) {
	response := &dap.SetExceptionBreakpointsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
}

func (h MonkeyHandler) OnConfigurationDoneRequest(request *dap.ConfigurationDoneRequest) {
	// This would be the place to check if the session was configured to
	// stop on entry and if that is the case, to issue a
	// stopped-on-breakpoint event. This being a mock implementation,
	// we "let" the program continue after sending a successful response.
	e := &dap.ThreadEvent{Event: *newEvent("thread"), Body: dap.ThreadEventBody{Reason: "started", ThreadId: 1}}
	h.session.send(e)
	response := &dap.ConfigurationDoneResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)

	se := &dap.StoppedEvent{
		Event: *newEvent("stopped"),
		Body:  dap.StoppedEventBody{Reason: "breakpoint", ThreadId: 1, AllThreadsStopped: true},
	}
	h.session.send(se)
	//h.session.doContinue()
}

func (h MonkeyHandler) OnContinueRequest(request *dap.ContinueRequest) {
	response := &dap.ContinueResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
	h.session.doContinue()
}

func (h MonkeyHandler) OnNextRequest(request *dap.NextRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "NextRequest is not yet supported"))
}

func (h MonkeyHandler) OnStepInRequest(request *dap.StepInRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "StepInRequest is not yet supported"))
}

func (h MonkeyHandler) OnStepOutRequest(request *dap.StepOutRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "StepOutRequest is not yet supported"))
}

func (h MonkeyHandler) OnStepBackRequest(request *dap.StepBackRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "StepBackRequest is not yet supported"))
}

func (h MonkeyHandler) OnReverseContinueRequest(request *dap.ReverseContinueRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "ReverseContinueRequest is not yet supported"))
}

func (h MonkeyHandler) OnRestartFrameRequest(request *dap.RestartFrameRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "RestartFrameRequest is not yet supported"))
}

func (h MonkeyHandler) OnGotoRequest(request *dap.GotoRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "GotoRequest is not yet supported"))
}

func (h MonkeyHandler) OnPauseRequest(request *dap.PauseRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "PauseRequest is not yet supported"))
}

func (h MonkeyHandler) OnStackTraceRequest(request *dap.StackTraceRequest) {
	log.Printf("VM: %v", h.Driver.VM)
	log.Printf("VM LOCS: %v", h.Driver.VM.LocationMap)
	response := &dap.StackTraceResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	vmLoc := h.Driver.VMLocation()
	response.Body = dap.StackTraceResponseBody{
		StackFrames: []dap.StackFrame{
			{
				Id:     1000,
				Source: &h.session.source,
				Line:   vmLoc,
				Column: 0,
				Name:   "main.main",
			},
		},
		//TotalFrames: 1,
		TotalFrames: 1,
	}
	h.session.send(response)
}

func (h MonkeyHandler) OnScopesRequest(request *dap.ScopesRequest) {
	response := &dap.ScopesResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body = dap.ScopesResponseBody{
		Scopes: []dap.Scope{
			{Name: "Local", VariablesReference: 1000, Expensive: false},
			{Name: "Global", VariablesReference: 1001, Expensive: true},
		},
	}
	h.session.send(response)
}

func (h MonkeyHandler) OnVariablesRequest(request *dap.VariablesRequest) {
	select {
	case <-h.session.stopDebug:
		return
	// simulate long-running processing to make this handler
	// respond to this request after the next request is received
	case <-time.After(100 * time.Millisecond):
		response := &dap.VariablesResponse{}
		response.Response = *newResponse(request.Seq, request.Command)
		response.Body = dap.VariablesResponseBody{
			Variables: []dap.Variable{{Name: "i", Value: "18434528", EvaluateName: "i", VariablesReference: 0}},
		}
		h.session.send(response)
	}
}

func (h MonkeyHandler) OnSetVariableRequest(request *dap.SetVariableRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "setVariableRequest is not yet supported"))
}

func (h MonkeyHandler) OnSetExpressionRequest(request *dap.SetExpressionRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SetExpressionRequest is not yet supported"))
}

func (h MonkeyHandler) OnSourceRequest(request *dap.SourceRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SourceRequest is not yet supported"))
}

func (h MonkeyHandler) OnThreadsRequest(request *dap.ThreadsRequest) {
	response := &dap.ThreadsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body = dap.ThreadsResponseBody{Threads: []dap.Thread{{Id: 1, Name: "main"}}}
	h.session.send(response)

}

func (h MonkeyHandler) OnTerminateThreadsRequest(request *dap.TerminateThreadsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "TerminateRequest is not yet supported"))
}

func (h MonkeyHandler) OnEvaluateRequest(request *dap.EvaluateRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "EvaluateRequest is not yet supported"))
}

func (h MonkeyHandler) OnStepInTargetsRequest(request *dap.StepInTargetsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "StepInTargetRequest is not yet supported"))
}

func (h MonkeyHandler) OnGotoTargetsRequest(request *dap.GotoTargetsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "GotoTargetRequest is not yet supported"))
}

func (h MonkeyHandler) OnCompletionsRequest(request *dap.CompletionsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "CompletionRequest is not yet supported"))
}

func (h MonkeyHandler) OnExceptionInfoRequest(request *dap.ExceptionInfoRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "ExceptionRequest is not yet supported"))
}

func (h MonkeyHandler) OnLoadedSourcesRequest(request *dap.LoadedSourcesRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "LoadedRequest is not yet supported"))
}

func (h MonkeyHandler) OnDataBreakpointInfoRequest(request *dap.DataBreakpointInfoRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "DataBreakpointInfoRequest is not yet supported"))
}

func (h MonkeyHandler) OnSetDataBreakpointsRequest(request *dap.SetDataBreakpointsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SetDataBreakpointsRequest is not yet supported"))
}

func (h MonkeyHandler) OnReadMemoryRequest(request *dap.ReadMemoryRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "ReadMemoryRequest is not yet supported"))
}

func (h MonkeyHandler) OnDisassembleRequest(request *dap.DisassembleRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "DisassembleRequest is not yet supported"))
}

func (h MonkeyHandler) OnCancelRequest(request *dap.CancelRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "CancelRequest is not yet supported"))
}

func (h MonkeyHandler) OnBreakpointLocationsRequest(request *dap.BreakpointLocationsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "BreakpointLocationsRequest is not yet supported"))
}
