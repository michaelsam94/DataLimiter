package datalimiter

type OSDeps struct{}

func (OSDeps) FindChrome() (string, error) {
	return FindChromeIn(ChromeCandidatePaths())
}

func (OSDeps) StateStore() StateStore {
	return ProgramDataStateStore{}
}
