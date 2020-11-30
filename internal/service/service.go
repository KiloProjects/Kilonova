package service

import "context"

type Service interface {
	Start(context.Context, map[string]string) error
}

var services map[string]Service

// Register should be used by the service in func init() to register
func Register(name string, s Service) {
	services[name] = s
}

// Get should be called by the main program to get the service to execute
func Get(name string) Service {
	return services[name]
}

func List() (names []string) {
	for k := range services {
		names = append(names, k)
	}
	return
}
