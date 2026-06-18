package plugin

import (
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/silveX89/woossh/model"
	sshpkg "github.com/silveX89/woossh/ssh"
)

// ─── Hook-Typen ────────────────────────────────────────────────────────────

// BeforeConnectHandler wird vor dem SSH-Connect aufgerufen.
// Gibt (modified entry, modified flags, err) zurück.
// err != nil bricht die Verbindung ab.
type BeforeConnectHandler func(entry model.HostEntry, flags sshpkg.Flags) (model.HostEntry, sshpkg.Flags, error)

// AfterConnectHandler wird nach dem SSH-Connect aufgerufen (exit code übergeben).
type AfterConnectHandler func(entry model.HostEntry, exitCode int)

// TUIKeyHandler wird für jeden Tastendruck in der TUI aufgerufen.
// Gibt (handled bool, cmd tea.Cmd) zurück.
// handled=true: Event gilt als konsumiert.
type TUIKeyHandler func(key string) (handled bool, cmd tea.Cmd)

// TUIExtraViewHandler liefert einen zusätzlich renderbaren String,
// der unter dem normalen View eingeblendet werden kann.
type TUIExtraViewHandler func() string

// CLICommandHandler wird aufgerufen, wenn woossh mit einem CLI-Argument gestartet wird.
// Gibt (handled bool) zurück. handled=true: main() macht os.Exit(0).
type CLICommandHandler func(args []string) (handled bool, exitCode int)

// HostLoadedHandler wird aufgerufen, nachdem hosts.csv/SSH-Config geladen wurde.
// Plugins können Hosts anreichern oder filtern.
type HostLoadedHandler func(hosts []model.HostEntry) []model.HostEntry

// ─── Hook-Registry-Interface ───────────────────────────────────────────────

// HookRegistry erlaubt Plugins das Anmelden von Handlern.
type HookRegistry interface {
	OnBeforeConnect(fn BeforeConnectHandler)
	OnAfterConnect(fn AfterConnectHandler)
	OnTUIKeyMsg(fn TUIKeyHandler)
	OnTUIExtraView(fn TUIExtraViewHandler)
	OnCLICommand(fn CLICommandHandler)
	OnHostLoaded(fn HostLoadedHandler)
}

// ─── Hook-Dispatcher (intern) ──────────────────────────────────────────────

// hookDispatcher verwaltet die registrierten Handler und implementiert HookRegistry.
type hookDispatcher struct {
	mu sync.RWMutex

	beforeConnect []BeforeConnectHandler
	afterConnect  []AfterConnectHandler
	tuiKey        []TUIKeyHandler
	tuiExtraView  []TUIExtraViewHandler
	cliCommand    []CLICommandHandler
	hostLoaded    []HostLoadedHandler
}

var _ HookRegistry = (*hookDispatcher)(nil)

func newHookDispatcher() *hookDispatcher {
	return &hookDispatcher{}
}

func (d *hookDispatcher) OnBeforeConnect(fn BeforeConnectHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.beforeConnect = append(d.beforeConnect, fn)
}

func (d *hookDispatcher) OnAfterConnect(fn AfterConnectHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.afterConnect = append(d.afterConnect, fn)
}

func (d *hookDispatcher) OnTUIKeyMsg(fn TUIKeyHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tuiKey = append(d.tuiKey, fn)
}

func (d *hookDispatcher) OnTUIExtraView(fn TUIExtraViewHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tuiExtraView = append(d.tuiExtraView, fn)
}

func (d *hookDispatcher) OnCLICommand(fn CLICommandHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cliCommand = append(d.cliCommand, fn)
}

func (d *hookDispatcher) OnHostLoaded(fn HostLoadedHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.hostLoaded = append(d.hostLoaded, fn)
}

// ─── Fire-Methoden (vom Manager aufgerufen) ────────────────────────────────

// FireBeforeConnect ruft alle registrierten Handler der Reihe nach auf.
func (d *hookDispatcher) FireBeforeConnect(entry model.HostEntry, flags sshpkg.Flags) (model.HostEntry, sshpkg.Flags, error) {
	d.mu.RLock()
	handlers := make([]BeforeConnectHandler, len(d.beforeConnect))
	copy(handlers, d.beforeConnect)
	d.mu.RUnlock()

	for _, fn := range handlers {
		var err error
		entry, flags, err = fn(entry, flags)
		if err != nil {
			return entry, flags, err
		}
	}
	return entry, flags, nil
}

// FireAfterConnect ruft alle registrierten AfterConnect-Handler auf.
func (d *hookDispatcher) FireAfterConnect(entry model.HostEntry, exitCode int) {
	d.mu.RLock()
	handlers := make([]AfterConnectHandler, len(d.afterConnect))
	copy(handlers, d.afterConnect)
	d.mu.RUnlock()

	for _, fn := range handlers {
		fn(entry, exitCode)
	}
}

// FireTUIKey ruft alle registrierten TUIKey-Handler auf.
// Stoppt beim ersten handled=true.
func (d *hookDispatcher) FireTUIKey(key string) (bool, tea.Cmd) {
	d.mu.RLock()
	handlers := make([]TUIKeyHandler, len(d.tuiKey))
	copy(handlers, d.tuiKey)
	d.mu.RUnlock()

	for _, fn := range handlers {
		handled, cmd := fn(key)
		if handled {
			return true, cmd
		}
	}
	return false, nil
}

// FireTUIExtraView sammelt alle Extra-View-Strings.
func (d *hookDispatcher) FireTUIExtraView() string {
	d.mu.RLock()
	handlers := make([]TUIExtraViewHandler, len(d.tuiExtraView))
	copy(handlers, d.tuiExtraView)
	d.mu.RUnlock()

	var result string
	for _, fn := range handlers {
		result += fn()
	}
	return result
}

// FireCLICommand ruft alle registrierten CLI-Handler auf.
func (d *hookDispatcher) FireCLICommand(args []string) (handled bool, exitCode int) {
	d.mu.RLock()
	handlers := make([]CLICommandHandler, len(d.cliCommand))
	copy(handlers, d.cliCommand)
	d.mu.RUnlock()

	for _, fn := range handlers {
		h, code := fn(args)
		if h {
			return true, code
		}
	}
	return false, 0
}

// FireHostLoaded ruft alle registrierten HostLoaded-Handler auf.
func (d *hookDispatcher) FireHostLoaded(hosts []model.HostEntry) []model.HostEntry {
	d.mu.RLock()
	handlers := make([]HostLoadedHandler, len(d.hostLoaded))
	copy(handlers, d.hostLoaded)
	d.mu.RUnlock()

	current := hosts
	for _, fn := range handlers {
		current = fn(current)
	}
	return current
}