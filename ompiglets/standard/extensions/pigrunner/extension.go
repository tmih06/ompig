package pigrunner

import (
	"fmt"

	sdk "github.com/tmih06/ompig/extensions/sdk"
	standardlogin "github.com/tmih06/ompig/ompiglets/standard/extensions/piglogin"
	"github.com/tmih06/ompig/ompiglets/standard/internal/termgame"
)

func Extension() *sdk.Extension {
	ext := sdk.New("pigrunner")
	ext.Command("runner", "Open Ompig Runner.", run)
	ext.Command("ompig-runner", "Open Ompig Runner.", run)
	return ext
}

func run(ctx sdk.Context, _ string) error {
	configHome, err := ctx.ConfigHome()
	if err != nil {
		return err
	}
	highScore := loadHighScore(configHome)
	component := newRunnerComponent(highScore, standardlogin.ActiveVariant(configHome), ctx.Height)
	result, err := ctx.Custom(component, termgame.Overlay("Ompig Runner"))
	if err != nil {
		return err
	}

	score, highScore := runnerScores(result, component.State())
	if err := saveHighScore(configHome, highScore); err != nil {
		return err
	}
	ctx.Notify(fmt.Sprintf("Ompig Runner score %d · high %d", score, highScore), "info")
	return nil
}

func runnerScores(result any, fallback runnerState) (score, highScore int) {
	score, highScore = fallback.Score, fallback.HighScore
	values, ok := result.(map[string]any)
	if !ok {
		return score, highScore
	}
	if value, ok := values["score"].(float64); ok && value >= 0 {
		score = int(value)
	}
	if value, ok := values["highScore"].(float64); ok && value >= 0 {
		highScore = int(value)
	}
	return score, max(highScore, score)
}
