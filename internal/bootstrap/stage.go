package bootstrap

const Stage = "stage-1-probe"

func ComponentStage(component string) string {
	if component == "" {
		return Stage
	}
	return component + ":" + Stage
}
