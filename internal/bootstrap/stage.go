package bootstrap

const Stage = "stage-0-bootstrap"

func ComponentStage(component string) string {
	if component == "" {
		return Stage
	}
	return component + ":" + Stage
}
