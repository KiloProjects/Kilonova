package language

func AI() Lang {
	return ai{}
}

type ai struct{}

func (ai) InternalName() string {
	return "ai"
}

func (ai) PrintableName() string {
	return "AI"
}

func (ai) Extensions() []string {
	return []string{".txt", ".csv", ".py", ".ai"}
}

func (ai) DefaultFilename() string {
	return "submit.csv"
}

func (ai) MOSSName() string {
	return "text"
}
