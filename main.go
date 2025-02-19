package main

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/termbox"
	"github.com/mum4k/termdash/widgets/linechart"
	"github.com/rivo/tview"
)

var (
	workDuration   time.Duration
	shortBreak     time.Duration
	longBreak      time.Duration
	pomodoroCount  = 0
	running        = false
	currentSession = "Work"
	remainingTime  time.Duration
)

func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func playSound() {
	switch runtime.GOOS {
	case "windows":
		exec.Command("cmd.exe", "/c", "echo \a").Run()
	default:
		exec.Command("play", "-nq", "synth", "0.5", "sin", "440").Run()
	}
}

func startTimer(app *tview.Application, timerLabel, progressBar *tview.TextView, cpuChart *linechart.LineChart) {
	running = true
	data := []float64{}
	for remainingTime > 0 && running {
		time.Sleep(time.Second)
		remainingTime -= time.Second
		progress := float64(remainingTime) / float64(getCurrentDuration())

		if len(data) >= 20 {
			data = data[1:]
		}
		data = append(data, (1-progress)*100)
		cpuChart.Series("Progress", data)

		app.QueueUpdateDraw(func() {
			timerLabel.SetText(fmt.Sprintf("[yellow]%s[-]\n[white]%s[-]", currentSession, formatDuration(remainingTime)))
			progressBar.SetText(fmt.Sprintf("[green]%.0f%% Complete[-]", (1-progress)*100))
		})
	}

	if running {
		playSound()
		pomodoroCount++
		if currentSession == "Work" {
			if pomodoroCount%4 == 0 {
				currentSession = "Long Break"
				remainingTime = longBreak
			} else {
				currentSession = "Short Break"
				remainingTime = shortBreak
			}
		} else {
			currentSession = "Work"
			remainingTime = workDuration
		}
		startTimer(app, timerLabel, progressBar, cpuChart)
	}
}

func getCurrentDuration() time.Duration {
	switch currentSession {
	case "Short Break":
		return shortBreak
	case "Long Break":
		return longBreak
	default:
		return workDuration
	}
}

func main() {
	work := flag.Int("work", 25, "Work duration in minutes")
	short := flag.Int("short", 5, "Short break duration in minutes")
	long := flag.Int("long", 15, "Long break duration in minutes")
	flag.Parse()

	workDuration = time.Duration(*work) * time.Minute
	shortBreak = time.Duration(*short) * time.Minute
	longBreak = time.Duration(*long) * time.Minute
	remainingTime = workDuration

	app := tview.NewApplication()
	timerLabel := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(fmt.Sprintf("[yellow]%s[-]\n[white]%s[-]", currentSession, formatDuration(remainingTime)))

	progressBar := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText("[green]0% Complete[-]")

	t, err := termbox.New()
	if err != nil {
		panic(err)
	}
	defer t.Close()

	cpuChart, err := linechart.New(
		linechart.YAxisFormattedValues(func(v float64) string {
			return fmt.Sprintf("%.0f%%", v)
		}),
		// linechart.SeriesOption(
		// 	// []float64{10, 20, 30, 40, 50},                           // ข้อมูลตัวอย่าง
		// 	// linechart.SeriesCellOpts(cell.FgColor(cell.ColorGreen)), // สีของเส้นกราฟ
		// ),
	)

	if err != nil {
		panic(err)
	}

	c, err := container.New(
		t,
		container.Border(linestyle.Light),
		container.BorderTitle(" Progress Chart "),
		container.PlaceWidget(cpuChart),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := termdash.Run(ctx, t, c); err != nil {
			panic(err)
		}
	}()

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(timerLabel, 5, 1, false).
		AddItem(progressBar, 3, 1, false)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 's':
			if !running {
				go startTimer(app, timerLabel, progressBar, cpuChart)
			}
		case 'p':
			running = false
		case 'r':
			running = false
			currentSession = "Work"
			remainingTime = workDuration
			app.QueueUpdateDraw(func() {
				timerLabel.SetText(fmt.Sprintf("[yellow]%s[-]\n[white]%s[-]", currentSession, formatDuration(remainingTime)))
				progressBar.SetText("[green]0% Complete[-]")
			})
		case 'q':
			app.Stop()
		}
		return event
	})

	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
