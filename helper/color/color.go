package color

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	Success = color.New(color.FgGreen).Add(color.Underline).SprintfFunc()
	Error   = color.New(color.FgRed).SprintfFunc()
	Warning = color.New(color.FgYellow).SprintfFunc()
	Info    = color.New(color.FgBlue).SprintfFunc()
	Notice  = color.New(color.FgCyan).SprintfFunc()
	Bold    = color.New(color.Bold).SprintfFunc()
)

// PrintColor
func PrintColor(text string, colorAttr color.Attribute) {
	_, _ = color.New(colorAttr).Println(text)
}

// PrintSuccess
func PrintSuccess(format string, a ...interface{}) {
	if len(a) == 0 {
		PrintColor(format, color.FgGreen)
	} else {
		PrintColor(fmt.Sprintf(format, a...), color.FgGreen)
	}
}

// PrintError
func PrintError(format string, a ...interface{}) {
	if len(a) == 0 {
		PrintColor(format, color.FgRed)
	} else {
		PrintColor(fmt.Sprintf(format, a...), color.FgRed)
	}
}

// PrintWarning
func PrintWarning(format string, a ...interface{}) {
	if len(a) == 0 {
		PrintColor(format, color.FgYellow)
	} else {
		PrintColor(fmt.Sprintf(format, a...), color.FgYellow)
	}
}

// PrintInfo
func PrintInfo(format string, a ...interface{}) {
	if len(a) == 0 {
		PrintColor(format, color.FgBlue)
	} else {
		PrintColor(fmt.Sprintf(format, a...), color.FgBlue)
	}
}

// PrintNotice
func PrintNotice(format string, a ...interface{}) {
	if len(a) == 0 {
		PrintColor(format, color.FgCyan)
	} else {
		PrintColor(fmt.Sprintf(format, a...), color.FgCyan)
	}
}

// ConfirmPrompt
func ConfirmPrompt(question string) bool {
	PrintWarning("%s", question)
	PrintNotice("Y: continue")
	PrintNotice("N: stop")

	_, err := color.New(color.FgHiYellow).Print("[y/n]:")
	if err != nil {
		PrintError("print error: %v", err)
		os.Exit(1)
	}
	var input string
	_, err = fmt.Scanln(&input)
	if err != nil {
		PrintError("input error: %v", err)
		os.Exit(1)
	}

	switch input {
	case "Y", "y":
		return true
	case "N", "n":
		PrintWarning("operation stopped")
	default:
		PrintWarning("invalid input. operation will be stopped by default")
	}

	return false
}

// ProgressBar
func ProgressBar(current, total int, message string) {
	if total <= 0 {
		total = 1
	}
	percentage := float64(current) / float64(total) * 100
	barLength := 40
	filled := int(float64(barLength) * float64(current) / float64(total))

	bar := ""
	for i := 0; i < barLength; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	fmt.Printf("\r%s [%s] %.1f%% (%d/%d)", message, bar, percentage, current, total)
	if current == total {
		fmt.Println()
	}
}

// PrintHeader
func PrintHeader(title string) {
	fmt.Println()
	PrintColor(strings.Repeat("=", len(title)+4), color.FgHiWhite)
	PrintColor("  "+title+"  ", color.BgBlue|color.FgWhite)
	PrintColor(strings.Repeat("=", len(title)+4), color.FgHiWhite)
	fmt.Println()
}

func PrintLogo() {
	logo := `
╔══════════════════════════════════════════════════════════════════════╗
║                                                                      ║
║    ██████╗ ███████╗ █████╗ ███╗   ██╗ ██████╗                        ║
║    ██╔══██╗██╔════╝██╔══██╗████╗  ██║██╔═══██╗                       ║
║    ██████╔╝█████╗  ███████║██╔██╗ ██║██║   ██║                       ║
║    ██╔══██╗██╔══╝  ██╔══██║██║╚██╗██║██║▄▄ ██║                       ║
║    ██████╔╝███████╗██║  ██║██║ ╚████║╚██████╔╝                       ║
║    ╚═════╝ ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚══▀▀═╝                        ║
║                                                                      ║
║                                   Queue Management System            ║
║                                                                      ║
╚══════════════════════════════════════════════════════════════════════╝
`
	_, _ = color.New(color.FgCyan).Println(logo)

}
