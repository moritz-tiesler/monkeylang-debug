package main

import (
	"time"

	"monkeylang-debug/driver"

	"github.com/google/go-dap"
)

type FakeHandler struct {
	session *fakeDebugSession
	Driver  driver.Driver
}

func (h *FakeHandler) SetSession(s *fakeDebugSession) {
	h.session = s
}

func (h FakeHandler) OnInitializeRequest(request *dap.InitializeRequest) {
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

func (h FakeHandler) OnLaunchRequest(request *dap.LaunchRequest) {
	// This is where a real debug adaptor would check the soundness of the
	// arguments (e.g. program from launch.json) and then use them to launch the
	// debugger and attach to the program.
	response := &dap.LaunchResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
}

func (h FakeHandler) OnAttachRequest(request *dap.AttachRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "AttachRequest is not yet supported"))
}

func (h FakeHandler) OnDisconnectRequest(request *dap.DisconnectRequest) {
	response := &dap.DisconnectResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
}

func (h FakeHandler) OnTerminateRequest(request *dap.TerminateRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "TerminateRequest is not yet supported"))
}

func (h FakeHandler) OnRestartRequest(request *dap.RestartRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "RestartRequest is not yet supported"))
}

func (h FakeHandler) OnSetBreakpointsRequest(request *dap.SetBreakpointsRequest) {
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

func (h FakeHandler) OnSetFunctionBreakpointsRequest(request *dap.SetFunctionBreakpointsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SetFunctionBreakpointsRequest is not yet supported"))
}

func (h FakeHandler) OnSetExceptionBreakpointsRequest(request *dap.SetExceptionBreakpointsRequest) {
	response := &dap.SetExceptionBreakpointsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
}

func (h FakeHandler) OnConfigurationDoneRequest(request *dap.ConfigurationDoneRequest) {
	// This would be the place to check if the session was configured to
	// stop on entry and if that is the case, to issue a
	// stopped-on-breakpoint event. This being a mock implementation,
	// we "let" the program continue after sending a successful response.
	e := &dap.ThreadEvent{Event: *newEvent("thread"), Body: dap.ThreadEventBody{Reason: "started", ThreadId: 1}}
	h.session.send(e)
	response := &dap.ConfigurationDoneResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
	h.session.doContinue()
}

func (h FakeHandler) OnContinueRequest(request *dap.ContinueRequest) {
	response := &dap.ContinueResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	h.session.send(response)
	h.session.doContinue()
}

func (h FakeHandler) OnNextRequest(request *dap.NextRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "NextRequest is not yet supported"))
}

func (h FakeHandler) OnStepInRequest(request *dap.StepInRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "StepInRequest is not yet supported"))
}

func (h FakeHandler) OnStepOutRequest(request *dap.StepOutRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "StepOutRequest is not yet supported"))
}

func (h FakeHandler) OnStepBackRequest(request *dap.StepBackRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "StepBackRequest is not yet supported"))
}

func (h FakeHandler) OnReverseContinueRequest(request *dap.ReverseContinueRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "ReverseContinueRequest is not yet supported"))
}

func (h FakeHandler) OnRestartFrameRequest(request *dap.RestartFrameRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "RestartFrameRequest is not yet supported"))
}

func (h FakeHandler) OnGotoRequest(request *dap.GotoRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "GotoRequest is not yet supported"))
}

func (h FakeHandler) OnPauseRequest(request *dap.PauseRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "PauseRequest is not yet supported"))
}

func (h FakeHandler) OnStackTraceRequest(request *dap.StackTraceRequest) {
	response := &dap.StackTraceResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body = dap.StackTraceResponseBody{
		StackFrames: []dap.StackFrame{
			{
				Id:     1000,
				Source: &dap.Source{Name: "hello.go", Path: "/Users/foo/go/src/hello/hello.go", SourceReference: 0},
				Line:   5,
				Column: 0,
				Name:   "main.main",
			},
		},
		TotalFrames: 1,
	}
	h.session.send(response)
}

func (h FakeHandler) OnScopesRequest(request *dap.ScopesRequest) {
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

func (h FakeHandler) OnVariablesRequest(request *dap.VariablesRequest) {
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

func (h FakeHandler) OnSetVariableRequest(request *dap.SetVariableRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "setVariableRequest is not yet supported"))
}

func (h FakeHandler) OnSetExpressionRequest(request *dap.SetExpressionRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SetExpressionRequest is not yet supported"))
}

func (h FakeHandler) OnSourceRequest(request *dap.SourceRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SourceRequest is not yet supported"))
}

func (h FakeHandler) OnThreadsRequest(request *dap.ThreadsRequest) {
	response := &dap.ThreadsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body = dap.ThreadsResponseBody{Threads: []dap.Thread{{Id: 1, Name: "main"}}}
	h.session.send(response)

}

func (h FakeHandler) OnTerminateThreadsRequest(request *dap.TerminateThreadsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "TerminateRequest is not yet supported"))
}

func (h FakeHandler) OnEvaluateRequest(request *dap.EvaluateRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "EvaluateRequest is not yet supported"))
}

func (h FakeHandler) OnStepInTargetsRequest(request *dap.StepInTargetsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "StepInTargetRequest is not yet supported"))
}

func (h FakeHandler) OnGotoTargetsRequest(request *dap.GotoTargetsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "GotoTargetRequest is not yet supported"))
}

func (h FakeHandler) OnCompletionsRequest(request *dap.CompletionsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "CompletionRequest is not yet supported"))
}

func (h FakeHandler) OnExceptionInfoRequest(request *dap.ExceptionInfoRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "ExceptionRequest is not yet supported"))
}

func (h FakeHandler) OnLoadedSourcesRequest(request *dap.LoadedSourcesRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "LoadedRequest is not yet supported"))
}

func (h FakeHandler) OnDataBreakpointInfoRequest(request *dap.DataBreakpointInfoRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "DataBreakpointInfoRequest is not yet supported"))
}

func (h FakeHandler) OnSetDataBreakpointsRequest(request *dap.SetDataBreakpointsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "SetDataBreakpointsRequest is not yet supported"))
}

func (h FakeHandler) OnReadMemoryRequest(request *dap.ReadMemoryRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "ReadMemoryRequest is not yet supported"))
}

func (h FakeHandler) OnDisassembleRequest(request *dap.DisassembleRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "DisassembleRequest is not yet supported"))
}

func (h FakeHandler) OnCancelRequest(request *dap.CancelRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "CancelRequest is not yet supported"))
}

func (h FakeHandler) OnBreakpointLocationsRequest(request *dap.BreakpointLocationsRequest) {
	h.session.send(newErrorResponse(request.Seq, request.Command, "BreakpointLocationsRequest is not yet supported"))
}
