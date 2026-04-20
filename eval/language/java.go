package language

import (
	"path"
	"slices"
	"strings"
)

type Java struct{}

func (j Java) InternalName() string {
	return "java"
}

func (j Java) PrintableName() string {
	return "Java"
}

func (j Java) Extensions() []string {
	return []string{".java", ".jar"}
}

func (j Java) DefaultFilename() string {
	return "Main.java"
}

func (j Java) MOSSName() string {
	return "java"
}

func (j Java) Compiled() bool {
	return true
}

func (j Java) CompileCommand(files []string) []string {
	// If jar, copy as-is
	if len(files) == 1 && strings.HasSuffix(files[0], ".jar") {
		return []string{"cp", files[0], "/box/output.jar"}
	}
	if len(files) == 0 {
		return []string{"err"}
	}

	return []string{"sh", "-c", "javac " + strings.Join(files, " ") + " && jar cfe output.jar " + strings.ReplaceAll(path.Base(files[0]), ".java", "") + " *.class"}
}

func (j Java) RunCommand(_ []string, _ int) []string {
	return slices.Concat([]string{"java", "-jar", "/box/output.jar"})
}

func (j Java) SourceName(userFilename string) string {
	if userFilename == "" {
		return "/box/Main.java"
	}
	return path.Join("/box/", userFilename)
}

func (j Java) CompiledName(_ string) string {
	return "/box/output.jar"
}

func (j Java) ExecuteName(_ string) string {
	return "/box/output.jar"
}

func (j Java) VersionCommand() []string {
	return []string{"javac", "-version"}
}

func (j Java) ParseVersion(version []byte) string {
	return string(version)
}

func (j Java) BuildEnv() map[string]string {
	return map[string]string{}
}

func (j Java) RunEnv() map[string]string {
	return map[string]string{}
}

func (j Java) Mounts() []Directory {
	return []Directory{{In: "/etc"}}
}

func (j Java) TimeLimitMultiplier() float64 {
	return 2.0
}

func (j Java) MemoryLimitMultiplier() float64 {
	return 1.0
}

func (j Java) SimilarLanguages() []string {
	return []string{}
}

func (j Java) Disabled() bool {
	return false
}

func (j Java) Lang() Lang {
	return j
}

func (j Java) GraderLang() GraderLang {
	return j
}
