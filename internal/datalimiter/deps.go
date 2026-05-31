package datalimiter

type OSDeps struct{}

func (OSDeps) FindChrome() (string, error) {
	return FindChromeIn(ChromeCandidatePaths())
}

func (OSDeps) ResolveApp(input string) (AllowedApp, error) {
	return ResolveExecutable(input)
}

func (OSDeps) StateStore() StateStore {
	return ProgramDataStateStore{}
}
