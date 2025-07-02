package cli

import (
	"fmt"
	"os"
	"time"
	
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProgressSpinner provides a simple spinner for long operations
type ProgressSpinner struct {
	spinner  spinner.Model
	message  string
	noColor  bool
	complete chan bool
	style    lipgloss.Style
}

// NewProgressSpinner creates a new progress spinner
func NewProgressSpinner(message string, noColor bool) *ProgressSpinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue
	
	return &ProgressSpinner{
		spinner:  s,
		message:  message,
		noColor:  noColor,
		complete: make(chan bool),
		style:    lipgloss.NewStyle().Foreground(lipgloss.Color("8")), // Gray for message
	}
}

// Start begins the spinner in a goroutine
func (p *ProgressSpinner) Start() {
	if p.noColor || os.Getenv("CI") != "" {
		// Just print the message without spinner in no-color mode
		fmt.Printf("%s...\n", p.message)
		return
	}
	
	// Create a simple program that just shows the spinner
	prog := &spinnerProgram{
		spinner:  p.spinner,
		message:  p.message,
		complete: p.complete,
		style:    p.style,
	}
	
	go func() {
		_ = tea.NewProgram(prog).Start()
	}()
	
	// Give the spinner a moment to start
	time.Sleep(50 * time.Millisecond)
}

// Stop stops the spinner
func (p *ProgressSpinner) Stop() {
	if !p.noColor && os.Getenv("CI") == "" {
		close(p.complete)
		// Give it a moment to clean up
		time.Sleep(50 * time.Millisecond)
	}
}

// spinnerProgram implements the tea.Model interface for the spinner
type spinnerProgram struct {
	spinner  spinner.Model
	message  string
	complete chan bool
	style    lipgloss.Style
}

func (s *spinnerProgram) Init() tea.Cmd {
	return tea.Batch(
		s.spinner.Tick,
		s.waitForComplete(),
	)
}

func (s *spinnerProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return s, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	case completeMsg:
		return s, tea.Quit
	}
	return s, nil
}

func (s *spinnerProgram) View() string {
	return fmt.Sprintf("%s %s", s.spinner.View(), s.style.Render(s.message))
}

func (s *spinnerProgram) waitForComplete() tea.Cmd {
	return func() tea.Msg {
		<-s.complete
		return completeMsg{}
	}
}

type completeMsg struct{}