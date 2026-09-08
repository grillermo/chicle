package chicle

import (
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// openTTY opens the controlling terminal. The UI draws here rather than to
// stdout so that stdout stays free to carry the picker's answer back to a
// calling shell — which is what makes `sel=$(mypicker)` work.
func openTTY() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}

// Run puts the list on the terminal and blocks until the user finishes. It
// returns the Result of whichever Action ended the picker, or "" if the user
// quit without choosing.
//
// Nothing is written to stdout. Print the result yourself if a shell wrapper
// is waiting for it.
func Run(cfg Config) (string, error) {
	tty, err := openTTY()
	if err != nil {
		return "", err
	}
	defer tty.Close()
	return runOn(cfg, tty, nil)
}

// runOn is Run with the terminal already opened, so the failure path is
// testable without one. openErr, when set, is returned untouched.
func runOn(cfg Config, tty *os.File, openErr error) (string, error) {
	if openErr != nil {
		return "", openErr
	}

	p := tea.NewProgram(New(cfg),
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)

	// A SIGTERM mid-draw would otherwise leave the terminal in raw mode with
	// no cursor. Quit through Bubble Tea so it runs its own restore.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		if _, ok := <-signals; ok {
			p.Quit()
		}
	}()

	final, err := p.Run()
	if err != nil {
		return "", err
	}
	return final.(Model).Result(), nil
}
