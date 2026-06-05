package language

func AI() Lang {
	return ai{}
}

type ai struct{}

func (A ai) InternalName() string {
	return "ai"
}

func (A ai) PrintableName() string {
	return "AI"
}

func (A ai) Extensions() []string {
	return []string{".txt", ".csv", ".py", ".ai"}
}

func (A ai) DefaultFilename() string {
	return "submit.csv"
}

func (A ai) MOSSName() string {
	return "text"
}
