package prompt

import (
	"bytes"
	"context"
	"time"

	logv2 "github.com/RafayLabs/rcloud-base/pkg/log"
)

var (
	_log = logv2.GetLogger()
)

// Executor is called when user input something text.
type Executor func(ctx context.Context, s string)

// ExitChecker is called after user input to check if prompt must stop and exit go-prompt Run loop.
// User input means: selecting/typing an entry, then, if said entry content matches the ExitChecker function criteria:
// - immediate exit (if breakline is false) without executor called
// - exit after typing <return> (meaning breakline is true), and the executor is called first, before exit.
// Exit means exit go-prompt (not the overall Go program)
type ExitChecker func(in string, breakline bool) bool

// Completer should return the suggest item from Document.
type Completer func(Document) []Suggest

// Prompt is core struct of go-prompt.
type Prompt struct {
	in                ConsoleParser
	buf               *Buffer
	renderer          *Render
	executor          Executor
	history           *History
	completion        *CompletionManager
	keyBindings       []KeyBind
	ASCIICodeBindings []ASCIICodeBind
	keyBindMode       KeyBindMode
	completionOnDown  bool
	exitChecker       ExitChecker
	skipTearDown      bool
}

// Exec is the struct contains user input context.
type Exec struct {
	input string
}

// Run starts prompt.
func (p *Prompt) Run(ctx context.Context) {
	p.skipTearDown = false
	// defer debug.Teardown()
	// debug.Log("start prompt")
	p.setUp()
	defer p.tearDown()

	if p.completion.showAtStart {
		p.completion.Update(*p.buf.Document())
	}

	p.renderer.Render(p.buf, p.completion)

	bufCh := make(chan []byte, 1)
	stopReadBufCh := make(chan struct{})
	go p.readBuffer(bufCh, stopReadBufCh)

promptLoop:
	for {
		select {
		case <-ctx.Done():
			stopReadBufCh <- struct{}{}
			break promptLoop

		case b := <-bufCh:
			if shouldExit, e := p.feed(b); shouldExit {
				p.renderer.BreakLine(p.buf)
				//p.in.Close()
				stopReadBufCh <- struct{}{}
				return
			} else if e != nil && e.input != "" {
				_log.Debugw("executing", "input", e.input)

				// Stop goroutine to run readBuffer function
				stopReadBufCh <- struct{}{}
				p.in.TearDown()

				_log.Debugw("executing", "input", e.input)
				p.executor(ctx, e.input)

				p.completion.Update(*p.buf.Document())
				p.renderer.Render(p.buf, p.completion)
				_log.Debugw("rendering prompt after executing")

				if p.exitChecker != nil && p.exitChecker(e.input, true) {
					p.skipTearDown = true
					return
				}

				_log.Debugw("starting read buffer again")
				p.in.Setup()
				go p.readBuffer(bufCh, stopReadBufCh)
			} else {
				p.completion.Update(*p.buf.Document())
				p.renderer.Render(p.buf, p.completion)
				_log.Debugw("rendering prompt without executing")
			}
		}
	}
}

func (p *Prompt) feed(b []byte) (shouldExit bool, exec *Exec) {
	key := GetKey(b)
	p.buf.lastKeyStroke = key
	// completion
	completing := p.completion.Completing()
	p.handleCompletionKeyBinding(key, completing)

	switch key {
	case Enter, ControlJ, ControlM:
		p.renderer.BreakLine(p.buf)
		_log.Debugw("enter key pressed", "buffer", p.buf.Text())
		exec = &Exec{input: p.buf.Text()}
		p.buf = NewBuffer()
		if exec.input != "" {
			p.history.Add(exec.input)
		}
	case ControlC:
		p.renderer.BreakLine(p.buf)
		p.buf = NewBuffer()
		p.history.Clear()
	case Up, ControlP:
		if !completing { // Don't use p.completion.Completing() because it takes double operation when switch to selected=-1.
			if newBuf, changed := p.history.Older(p.buf); changed {
				p.buf = newBuf
			}
		}
	case Down, ControlN:
		if !completing { // Don't use p.completion.Completing() because it takes double operation when switch to selected=-1.
			if newBuf, changed := p.history.Newer(p.buf); changed {
				p.buf = newBuf
			}
			return
		}
	case ControlD:
		if p.buf.Text() == "" {
			shouldExit = true
			return
		}
	case NotDefined:
		if p.handleASCIICodeBinding(b) {
			return
		}
		p.buf.InsertText(string(b), false, true)
	}

	shouldExit = p.handleKeyBinding(key)
	return
}

func (p *Prompt) handleCompletionKeyBinding(key Key, completing bool) {
	switch key {
	case Down:
		if completing || p.completionOnDown {
			p.completion.Next()
		}
	case Tab, ControlI:
		p.completion.Next()
	case Up:
		if completing {
			p.completion.Previous()
		}
	case BackTab:
		p.completion.Previous()
	default:
		if s, ok := p.completion.GetSelectedSuggestion(); ok {
			w := p.buf.Document().GetWordBeforeCursorUntilSeparator(p.completion.wordSeparator)
			if w != "" {
				p.buf.DeleteBeforeCursor(len([]rune(w)))
			}
			p.buf.InsertText(s.Text, false, true)
		}
		p.completion.Reset()
	}
}

func (p *Prompt) handleKeyBinding(key Key) bool {
	shouldExit := false
	for i := range commonKeyBindings {
		kb := commonKeyBindings[i]
		if kb.Key == key {
			kb.Fn(p.buf)
		}
	}

	if p.keyBindMode == EmacsKeyBind {
		for i := range emacsKeyBindings {
			kb := emacsKeyBindings[i]
			if kb.Key == key {
				kb.Fn(p.buf)
			}
		}
	}

	// Custom key bindings
	for i := range p.keyBindings {
		kb := p.keyBindings[i]
		if kb.Key == key {
			kb.Fn(p.buf)
		}
	}
	if p.exitChecker != nil && p.exitChecker(p.buf.Text(), false) {
		shouldExit = true
	}
	return shouldExit
}

func (p *Prompt) handleASCIICodeBinding(b []byte) bool {
	checked := false
	for _, kb := range p.ASCIICodeBindings {
		if bytes.Equal(kb.ASCIICode, b) {
			kb.Fn(p.buf)
			checked = true
		}
	}
	return checked
}

func (p *Prompt) readBuffer(bufCh chan []byte, stopCh chan struct{}) {
	_log.Debugw("start reading buffer")
readLoop:
	for {
		select {
		case <-stopCh:
			_log.Debugw("stop reading buffer")
			return
		default:
			b, err := p.in.Read()
			if err != nil {
				_log.Infow("unable to read buffer", "error", err)
				break readLoop
			}

			if len(b) > 0 && !(len(b) == 1 && b[0] == 0) {
				bufCh <- b
			}

		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (p *Prompt) setUp() {
	p.in.Setup()
	p.renderer.Setup()
	p.renderer.UpdateWinSize(p.in.GetWinSize())
}

func (p *Prompt) tearDown() {
	// if !p.skipTearDown {
	// 	debug.AssertNoError(p.in.TearDown())
	// }
	p.renderer.TearDown()
}

// RunPreset starts preset command.
func (p *Prompt) RunPreset(ctx context.Context, command string) {
	p.skipTearDown = false
	// defer debug.Teardown()
	// debug.Log("start prompt")
	p.setUp()
	defer p.tearDown()

	if p.completion.showAtStart {
		p.completion.Update(*p.buf.Document())
	}

	p.executor(ctx, command)
	p.completion.Update(*p.buf.Document())

	p.renderer.Render(p.buf, p.completion)

	bufCh := make(chan []byte, 1)
	stopReadBufCh := make(chan struct{})
	go p.readBuffer(bufCh, stopReadBufCh)

presetLoop:
	for {
		select {
		case <-ctx.Done():
			stopReadBufCh <- struct{}{}
			break presetLoop

		case b := <-bufCh:
			if shouldExit, e := p.feed(b); shouldExit {
				p.renderer.BreakLine(p.buf)
				//p.in.Close()
				stopReadBufCh <- struct{}{}
				return
			} else if e != nil && e.input != "" {
				_log.Debugw("executing", "input", e.input)

				// Stop goroutine to run readBuffer function
				stopReadBufCh <- struct{}{}
				p.in.TearDown()

				_log.Debugw("executing", "input", e.input)
				p.executor(ctx, e.input)

				p.completion.Update(*p.buf.Document())
				p.renderer.Render(p.buf, p.completion)
				_log.Debugw("rendering prompt after executing")

				if p.exitChecker != nil && p.exitChecker(e.input, true) {
					p.skipTearDown = true
					return
				}

				_log.Debugw("starting read buffer again")
				p.in.Setup()
				go p.readBuffer(bufCh, stopReadBufCh)
			} else {
				p.completion.Update(*p.buf.Document())
				p.renderer.Render(p.buf, p.completion)
				_log.Debugw("rendering prompt without executing")
			}
		}
	}
}
