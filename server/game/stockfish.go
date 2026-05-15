package game

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultStockfishPath = "stockfish/stockfish"

type StockfishEngine struct {
	path      string
	level     int
	thinkTime time.Duration
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Reader
	mu        sync.Mutex
}

func NewStockfishEngine(path string, level int) (*StockfishEngine, error) {
	if path == "" {
		path = defaultStockfishPath
	}

	level = normalizeAILevel(level)
	engine := &StockfishEngine{
		path:      path,
		level:     level,
		thinkTime: stockfishThinkTime(level),
	}

	if err := engine.start(); err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *StockfishEngine) start() error {
	e.cmd = exec.Command(e.path)

	stdin, err := e.cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := e.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	e.cmd.Stderr = io.Discard

	if err := e.cmd.Start(); err != nil {
		return err
	}

	e.stdin = stdin
	e.stdout = bufio.NewReader(stdout)

	if err := e.writeLine("uci"); err != nil {
		e.Close()
		return err
	}
	if err := e.readUntil("uciok", 2*time.Second); err != nil {
		e.Close()
		return err
	}
	if err := e.configureStrength(); err != nil {
		e.Close()
		return err
	}
	if err := e.ready(); err != nil {
		e.Close()
		return err
	}

	return nil
}

func (e *StockfishEngine) configureStrength() error {
	commands := []string{
		"setoption name Skill Level value " + strconv.Itoa(stockfishSkillLevel(e.level)),
		"setoption name UCI_LimitStrength value true",
		"setoption name UCI_Elo value " + strconv.Itoa(stockfishELO(e.level)),
	}

	for _, command := range commands {
		if err := e.writeLine(command); err != nil {
			return err
		}
	}
	return nil
}

func (e *StockfishEngine) ready() error {
	if err := e.writeLine("isready"); err != nil {
		return err
	}
	return e.readUntil("readyok", 2*time.Second)
}

func (e *StockfishEngine) BestMove(ctx context.Context, fen string) (string, string, string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cmd == nil || e.stdin == nil || e.stdout == nil {
		return "", "", "", errors.New("stockfish is not running")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	moveCtx, cancel := context.WithTimeout(ctx, e.thinkTime+2*time.Second)
	defer cancel()

	if err := e.writeLine("position fen " + fen); err != nil {
		return "", "", "", err
	}
	// if err := e.writeLine("go movetime " + strconv.Itoa(int(e.thinkTime.Milliseconds()))); err != nil {
	// 	return "", "", "", err
	// }

	if err := e.writeLine(
		"go depth " + strconv.Itoa(stockfishDepth(e.level)),
	); err != nil {
		return "", "", "", err
	}

	lineCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		for {
			line, err := e.stdout.ReadString('\n')
			if err != nil {
				errCh <- err
				return
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "bestmove ") {
				lineCh <- line
				return
			}
		}
	}()

	select {
	case <-moveCtx.Done():
		e.stopLocked()
		return "", "", "", moveCtx.Err()
	case err := <-errCh:
		e.stopLocked()
		return "", "", "", err
	case line := <-lineCh:
		return parseBestMove(line)
	}
}

func (e *StockfishEngine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.stopLocked()
}

func (e *StockfishEngine) stopLocked() {
	if e.stdin != nil {
		_ = e.writeLine("quit")
		_ = e.stdin.Close()
		e.stdin = nil
	}
	if e.cmd != nil && e.cmd.Process != nil {
		done := make(chan error, 1)
		go func() {
			done <- e.cmd.Wait()
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			_ = e.cmd.Process.Kill()
			<-done
		}
	}
	e.cmd = nil
	e.stdout = nil
}

func (e *StockfishEngine) writeLine(command string) error {
	_, err := io.WriteString(e.stdin, command+"\n")
	return err
}

func (e *StockfishEngine) readUntil(expected string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		line, err := e.stdout.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.TrimSpace(line) == expected {
			return nil
		}
	}
	return fmt.Errorf("stockfish did not return %s", expected)
}

func parseBestMove(line string) (string, string, string, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", "", errors.New("invalid stockfish bestmove response")
	}

	move := fields[1]
	if move == "(none)" || len(move) < 4 {
		return "", "", "", errors.New("stockfish returned no legal move")
	}

	promotion := ""
	if len(move) >= 5 {
		promotion = move[4:5]
	}

	return move[0:2], move[2:4], promotion, nil
}

func stockfishSkillLevel(level int) int {
	level = normalizeAILevel(level)
	return (level - 1) * 20 / 9
}

func stockfishELO(level int) int {
	level = normalizeAILevel(level)
	switch level {
	case 1:
		log.Println("masuk ke 1 <<<<<<<<")
		return 100
	case 2:
		return 750
	case 3:
		return 900
	case 4:
		return 1050
	case 5:
		return 1200
	case 6:
		return 1400
	case 7:
		return 1600
	case 8:
		return 1900
	case 9:
		return 2200
	case 10:
		return 2500
	default:
		return 1000
	}
}

func stockfishThinkTime(level int) time.Duration {
	level = normalizeAILevel(level)
	switch level {
	case 1:
		return 300 * time.Millisecond
	case 2:
		return 500 * time.Millisecond
	case 3:
		return 700 * time.Millisecond
	case 4:
		return 1 * time.Second
	case 5:
		return 1200 * time.Millisecond
	case 6:
		return 1500 * time.Millisecond
	case 7:
		return 2 * time.Second
	case 8:
		return 2500 * time.Millisecond
	case 9:
		return 3 * time.Second
	case 10:
		return 4 * time.Second
	default:
		return 1 * time.Second
	}
}

func stockfishDepth(level int) int {
	level = normalizeAILevel(level)

	switch level {
	case 1:
		return 1
	case 2:
		return 1
	case 3:
		return 2
	case 4:
		return 2
	case 5:
		return 3
	case 6:
		return 4
	case 7:
		return 5
	case 8:
		return 6
	case 9:
		return 8
	case 10:
		return 10
	default:
		return 2
	}
}
