package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/go-dap"
)

// StdioReadWriteCloser is a ReadWriteCloser for reading from stdin and writing to stdout
// If Log is true, all incoming and outgoing data is logged
type StdioReadWriteCloser struct {
}

func (s StdioReadWriteCloser) Read(p []byte) (n int, err error) {
	return os.Stdin.Read(p)
}

func (s StdioReadWriteCloser) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

// Close closes stdin and stdout
func (s StdioReadWriteCloser) Close() error {
	err := os.Stdin.Close()
	if err != nil {
		return err
	}
	return os.Stdout.Close()
}

type fakeDebugSession struct {
	// rw is used to read requests and write events/responses
	rw *bufio.ReadWriter

	// sendQueue is used to capture messages from multiple request
	// processing goroutines while writing them to the client connection
	// from a single goroutine via sendFromQueue. We must keep track of
	// the multiple channel senders with a wait group to make sure we do
	// not close this channel prematurely. Closing this channel will signal
	// the sendFromQueue goroutine that it can exit.
	sendQueue chan dap.Message
	sendWg    sync.WaitGroup

	// stopDebug is used to notify long-running handlers to stop processing.
	stopDebug chan struct{}

	handler Handler
	// bpSet is a counter of the remaining breakpoints that the debug
	// session is yet to stop at before the program terminates.
	bpSet       int
	bpSetMux    sync.Mutex
	breakPoints []dap.SourceBreakpoint
	source      dap.Source
}

func StartSession(conn io.ReadWriteCloser) {
	debugSession := fakeDebugSession{
		rw:        bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		sendQueue: make(chan dap.Message),
		stopDebug: make(chan struct{}),
		//cdebuglog:  debuglog,
	}
	debugSession.handler = &debugSession
	//debugSession.handler.SetSession(&debugSession)

	go debugSession.sendFromQueue()

	for {
		err := debugSession.handleRequest()
		// TODO(polina): check for connection vs decoding error?
		if err != nil {
			if err == io.EOF {
				log.Println("No more data to read:", err)
				break
			}
			// There maybe more messages to process, but
			// we will start with the strict behavior of only accepting
			// expected inputs.
			log.Fatal("Server error: ", err)
		}
	}

	close(debugSession.stopDebug)
	debugSession.sendWg.Wait()
	close(debugSession.sendQueue)
	conn.Close()
}

func (ds *fakeDebugSession) send(message dap.Message) {
	ds.sendQueue <- message
}

func (ds *fakeDebugSession) sendFromQueue() {
	for message := range ds.sendQueue {
		dap.WriteProtocolMessage(ds.rw.Writer, message)
		log.Printf("Message sent\n\t%#v\n", message)
		ds.rw.Flush()
	}
}

func (ds *fakeDebugSession) doContinue() {
	var e dap.Message
	ds.bpSetMux.Lock()
	if ds.bpSet == 0 {
		// Pretend that the program is running.
		// The delay will allow for all in-flight responses
		// to be sent before termination.
		time.Sleep(1000 * time.Millisecond)
		e = &dap.TerminatedEvent{
			Event: *newEvent("terminated"),
		}
	} else {
		e = &dap.StoppedEvent{
			Event: *newEvent("stopped"),
			Body:  dap.StoppedEventBody{Reason: "breakpoint", ThreadId: 1, AllThreadsStopped: true},
		}
		ds.breakPoints = ds.breakPoints[1:]
		ds.bpSet--
	}
	ds.bpSetMux.Unlock()
	ds.send(e)
}

func (ds *fakeDebugSession) handleRequest() error {
	log.Println("Reading request...")
	request, err := dap.ReadProtocolMessage(ds.rw.Reader)
	if err != nil {
		return err
	}
	log.Printf("Received request\n\t%#v\n", request)
	log.Printf("%v", ds.breakPoints)
	ds.sendWg.Add(1)
	go func() {
		ds.dispatchRequest(request)
		ds.sendWg.Done()
	}()
	return nil
}

// dispatchRequest launches a new goroutine to process each request
// and send back events and responses.
func (ds *fakeDebugSession) dispatchRequest(request dap.Message) {
	switch request := request.(type) {
	case *dap.InitializeRequest:
		ds.OnInitializeRequest(request)
	case *dap.LaunchRequest:
		ds.OnLaunchRequest(request)
	case *dap.AttachRequest:
		ds.OnAttachRequest(request)
	case *dap.DisconnectRequest:
		ds.OnDisconnectRequest(request)
	case *dap.TerminateRequest:
		ds.OnTerminateRequest(request)
	case *dap.RestartRequest:
		ds.OnRestartRequest(request)
	case *dap.SetBreakpointsRequest:
		ds.OnSetBreakpointsRequest(request)
	case *dap.SetFunctionBreakpointsRequest:
		ds.OnSetFunctionBreakpointsRequest(request)
	case *dap.SetExceptionBreakpointsRequest:
		ds.OnSetExceptionBreakpointsRequest(request)
	case *dap.ConfigurationDoneRequest:
		ds.OnConfigurationDoneRequest(request)
	case *dap.ContinueRequest:
		ds.OnContinueRequest(request)
	case *dap.NextRequest:
		ds.OnNextRequest(request)
	case *dap.StepInRequest:
		ds.OnStepInRequest(request)
	case *dap.StepOutRequest:
		ds.OnStepOutRequest(request)
	case *dap.StepBackRequest:
		ds.OnStepBackRequest(request)
	case *dap.ReverseContinueRequest:
		ds.OnReverseContinueRequest(request)
	case *dap.RestartFrameRequest:
		ds.OnRestartFrameRequest(request)
	case *dap.GotoRequest:
		ds.OnGotoRequest(request)
	case *dap.PauseRequest:
		ds.OnPauseRequest(request)
	case *dap.StackTraceRequest:
		ds.OnStackTraceRequest(request)
	case *dap.ScopesRequest:
		ds.OnScopesRequest(request)
	case *dap.VariablesRequest:
		ds.OnVariablesRequest(request)
	case *dap.SetVariableRequest:
		ds.OnSetVariableRequest(request)
	case *dap.SetExpressionRequest:
		ds.OnSetExpressionRequest(request)
	case *dap.SourceRequest:
		ds.OnSourceRequest(request)
	case *dap.ThreadsRequest:
		ds.OnThreadsRequest(request)
	case *dap.TerminateThreadsRequest:
		ds.OnTerminateThreadsRequest(request)
	case *dap.EvaluateRequest:
		ds.OnEvaluateRequest(request)
	case *dap.StepInTargetsRequest:
		ds.OnStepInTargetsRequest(request)
	case *dap.GotoTargetsRequest:
		ds.OnGotoTargetsRequest(request)
	case *dap.CompletionsRequest:
		ds.OnCompletionsRequest(request)
	case *dap.ExceptionInfoRequest:
		ds.OnExceptionInfoRequest(request)
	case *dap.LoadedSourcesRequest:
		ds.OnLoadedSourcesRequest(request)
	case *dap.DataBreakpointInfoRequest:
		ds.OnDataBreakpointInfoRequest(request)
	case *dap.SetDataBreakpointsRequest:
		ds.OnSetDataBreakpointsRequest(request)
	case *dap.ReadMemoryRequest:
		ds.OnReadMemoryRequest(request)
	case *dap.DisassembleRequest:
		ds.OnDisassembleRequest(request)
	case *dap.CancelRequest:
		ds.OnCancelRequest(request)
	case *dap.BreakpointLocationsRequest:
		ds.OnBreakpointLocationsRequest(request)
	default:
		log.Fatalf("Unable to process %#v", request)
	}
}

// -----------------------------------------------------------------------
// Request Handlers
//
// Below is a dummy implementation of the request handlers.
// They take no action, but just return dummy responses.
// A real debug adaptor would call the debugger methods here
// and use their results to populate each response.

func (ds *fakeDebugSession) OnInitializeRequest(request *dap.InitializeRequest) {
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
	ds.send(e)
	ds.send(response)
}

func (ds *fakeDebugSession) OnLaunchRequest(request *dap.LaunchRequest) {
	// This is where a real debug adaptor would check the soundness of the
	// arguments (e.g. program from launch.json) and then use them to launch the
	// debugger and attach to the program.
	response := &dap.LaunchResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	ds.send(response)
}

func (ds *fakeDebugSession) OnAttachRequest(request *dap.AttachRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "AttachRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnDisconnectRequest(request *dap.DisconnectRequest) {
	response := &dap.DisconnectResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	ds.send(response)
}

func (ds *fakeDebugSession) OnTerminateRequest(request *dap.TerminateRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "TerminateRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnRestartRequest(request *dap.RestartRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "RestartRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnSetBreakpointsRequest(request *dap.SetBreakpointsRequest) {
	source := request.Arguments.Source
	ds.source = source
	ds.breakPoints = request.Arguments.Breakpoints
	response := &dap.SetBreakpointsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body.Breakpoints = make([]dap.Breakpoint, len(request.Arguments.Breakpoints))
	for i, b := range request.Arguments.Breakpoints {
		response.Body.Breakpoints[i].Line = b.Line
		response.Body.Breakpoints[i].Verified = true
		ds.bpSetMux.Lock()
		ds.bpSet++
		ds.bpSetMux.Unlock()
	}
	ds.send(response)
}

func (ds *fakeDebugSession) OnSetFunctionBreakpointsRequest(request *dap.SetFunctionBreakpointsRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "SetFunctionBreakpointsRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnSetExceptionBreakpointsRequest(request *dap.SetExceptionBreakpointsRequest) {
	response := &dap.SetExceptionBreakpointsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	ds.send(response)
}

func (ds *fakeDebugSession) OnConfigurationDoneRequest(request *dap.ConfigurationDoneRequest) {
	// This would be the place to check if the session was configured to
	// stop on entry and if that is the case, to issue a
	// stopped-on-breakpoint event. This being a mock implementation,
	// we "let" the program continue after sending a successful response.
	e := &dap.ThreadEvent{Event: *newEvent("thread"), Body: dap.ThreadEventBody{Reason: "started", ThreadId: 1}}
	ds.send(e)
	response := &dap.ConfigurationDoneResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	ds.send(response)

	se := &dap.StoppedEvent{
		Event: *newEvent("stopped"),
		Body:  dap.StoppedEventBody{Reason: "breakpoint", ThreadId: 1, AllThreadsStopped: true},
	}
	ds.send(se)
	//ds.doContinue()
}

func (ds *fakeDebugSession) OnContinueRequest(request *dap.ContinueRequest) {
	response := &dap.ContinueResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	ds.send(response)
	ds.doContinue()
}

func (ds *fakeDebugSession) OnNextRequest(request *dap.NextRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "NextRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnStepInRequest(request *dap.StepInRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "StepInRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnStepOutRequest(request *dap.StepOutRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "StepOutRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnStepBackRequest(request *dap.StepBackRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "StepBackRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnReverseContinueRequest(request *dap.ReverseContinueRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "ReverseContinueRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnRestartFrameRequest(request *dap.RestartFrameRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "RestartFrameRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnGotoRequest(request *dap.GotoRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "GotoRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnPauseRequest(request *dap.PauseRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "PauseRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnStackTraceRequest(request *dap.StackTraceRequest) {
	currentBp := ds.breakPoints[0]
	response := &dap.StackTraceResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body = dap.StackTraceResponseBody{
		StackFrames: []dap.StackFrame{
			{
				Id:     1000,
				Source: &ds.source,
				Line:   currentBp.Line,
				Column: 0,
				Name:   "main.main",
			},
		},
		//TotalFrames: 1,
		TotalFrames: 1,
	}
	ds.send(response)
}

func (ds *fakeDebugSession) OnScopesRequest(request *dap.ScopesRequest) {
	response := &dap.ScopesResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body = dap.ScopesResponseBody{
		Scopes: []dap.Scope{
			{Name: "Local", VariablesReference: 1000, Expensive: false},
			{Name: "Global", VariablesReference: 1001, Expensive: true},
		},
	}
	ds.send(response)
}

func (ds *fakeDebugSession) OnVariablesRequest(request *dap.VariablesRequest) {
	select {
	case <-ds.stopDebug:
		return
	// simulate long-running processing to make this handler
	// respond to this request after the next request is received
	case <-time.After(100 * time.Millisecond):
		response := &dap.VariablesResponse{}
		response.Response = *newResponse(request.Seq, request.Command)
		response.Body = dap.VariablesResponseBody{
			Variables: []dap.Variable{{Name: "i", Value: "18434528", EvaluateName: "i", VariablesReference: 0}},
		}
		ds.send(response)
	}
}

func (ds *fakeDebugSession) OnSetVariableRequest(request *dap.SetVariableRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "setVariableRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnSetExpressionRequest(request *dap.SetExpressionRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "SetExpressionRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnSourceRequest(request *dap.SourceRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "SourceRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnThreadsRequest(request *dap.ThreadsRequest) {
	response := &dap.ThreadsResponse{}
	response.Response = *newResponse(request.Seq, request.Command)
	response.Body = dap.ThreadsResponseBody{Threads: []dap.Thread{{Id: 1, Name: "main"}}}
	ds.send(response)

}

func (ds *fakeDebugSession) OnTerminateThreadsRequest(request *dap.TerminateThreadsRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "TerminateRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnEvaluateRequest(request *dap.EvaluateRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "EvaluateRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnStepInTargetsRequest(request *dap.StepInTargetsRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "StepInTargetRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnGotoTargetsRequest(request *dap.GotoTargetsRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "GotoTargetRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnCompletionsRequest(request *dap.CompletionsRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "CompletionRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnExceptionInfoRequest(request *dap.ExceptionInfoRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "ExceptionRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnLoadedSourcesRequest(request *dap.LoadedSourcesRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "LoadedRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnDataBreakpointInfoRequest(request *dap.DataBreakpointInfoRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "DataBreakpointInfoRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnSetDataBreakpointsRequest(request *dap.SetDataBreakpointsRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "SetDataBreakpointsRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnReadMemoryRequest(request *dap.ReadMemoryRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "ReadMemoryRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnDisassembleRequest(request *dap.DisassembleRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "DisassembleRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnCancelRequest(request *dap.CancelRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "CancelRequest is not yet supported"))
}

func (ds *fakeDebugSession) OnBreakpointLocationsRequest(request *dap.BreakpointLocationsRequest) {
	ds.send(newErrorResponse(request.Seq, request.Command, "BreakpointLocationsRequest is not yet supported"))
}

func newEvent(event string) *dap.Event {
	return &dap.Event{
		ProtocolMessage: dap.ProtocolMessage{
			Seq:  0,
			Type: "event",
		},
		Event: event,
	}
}

func newResponse(requestSeq int, command string) *dap.Response {
	return &dap.Response{
		ProtocolMessage: dap.ProtocolMessage{
			Seq:  0,
			Type: "response",
		},
		Command:    command,
		RequestSeq: requestSeq,
		Success:    true,
	}
}

func newErrorResponse(requestSeq int, command string, message string) *dap.ErrorResponse {
	er := &dap.ErrorResponse{}
	er.Response = *newResponse(requestSeq, command)
	er.Success = false
	er.Message = "unsupported"
	er.Body.Error.Format = message
	er.Body.Error.Id = 12345
	return er
}

type Session struct {
	// rw is used to read requests and write events/responses
	rw *bufio.ReadWriter

	// sendQueue is used to capture messages from multiple request
	// processing goroutines while writing them to the client connection
	// from a single goroutine via sendFromQueue. We must keep track of
	// the multiple channel senders with a wait group to make sure we do
	// not close this channel prematurely. Closing this channel will signal
	// the sendFromQueue goroutine that it can exit.
	sendQueue chan dap.Message
	sendWg    sync.WaitGroup

	// stopDebug is used to notify long-running handlers to stop processing.
	stopDebug chan struct{}

	// handler does the actual handling of the requests
	handler Handler

	debuglog bool
}

func (ds *fakeDebugSession) SetSession(s *fakeDebugSession) {}

type Handler interface {
	SetSession(s *fakeDebugSession)
	OnInitializeRequest(arguments *dap.InitializeRequest)
	OnLaunchRequest(arguments *dap.LaunchRequest)
	OnDisconnectRequest(arguments *dap.DisconnectRequest)
	OnTerminateRequest(arguments *dap.TerminateRequest)
	OnSetBreakpointsRequest(arguments *dap.SetBreakpointsRequest)
	OnSetFunctionBreakpointsRequest(arguments *dap.SetFunctionBreakpointsRequest)
	OnSetExceptionBreakpointsRequest(arguments *dap.SetExceptionBreakpointsRequest)
	OnConfigurationDoneRequest(arguments *dap.ConfigurationDoneRequest)
	OnContinueRequest(arguments *dap.ContinueRequest)
	OnNextRequest(arguments *dap.NextRequest)
	OnStepInRequest(arguments *dap.StepInRequest)
	OnStepOutRequest(arguments *dap.StepOutRequest)
	OnStepBackRequest(arguments *dap.StepBackRequest)
	OnReverseContinueRequest(arguments *dap.ReverseContinueRequest)
	OnRestartFrameRequest(arguments *dap.RestartFrameRequest)
	OnGotoRequest(arguments *dap.GotoRequest)
	OnPauseRequest(arguments *dap.PauseRequest)
	OnStackTraceRequest(arguments *dap.StackTraceRequest)
	OnScopesRequest(arguments *dap.ScopesRequest)
	OnVariablesRequest(arguments *dap.VariablesRequest)
	OnSetVariableRequest(arguments *dap.SetVariableRequest)
	OnSetExpressionRequest(arguments *dap.SetExpressionRequest)
	OnSourceRequest(arguments *dap.SourceRequest)
	OnThreadsRequest(arguments *dap.ThreadsRequest)
	OnTerminateThreadsRequest(arguments *dap.TerminateThreadsRequest)
	OnEvaluateRequest(arguments *dap.EvaluateRequest)
	OnStepInTargetsRequest(arguments *dap.StepInTargetsRequest)
	OnGotoTargetsRequest(arguments *dap.GotoTargetsRequest)
	OnCompletionsRequest(arguments *dap.CompletionsRequest)
	OnExceptionInfoRequest(arguments *dap.ExceptionInfoRequest)
	OnLoadedSourcesRequest(arguments *dap.LoadedSourcesRequest)
	OnDataBreakpointInfoRequest(arguments *dap.DataBreakpointInfoRequest)
	OnSetDataBreakpointsRequest(arguments *dap.SetDataBreakpointsRequest)
	OnReadMemoryRequest(arguments *dap.ReadMemoryRequest)
	OnDisassembleRequest(arguments *dap.DisassembleRequest)
	OnCancelRequest(arguments *dap.CancelRequest)
	OnBreakpointLocationsRequest(arguments *dap.BreakpointLocationsRequest)
}
