package main

// GreetService is a demo service to prove the Go<->Frontend bridge works.
// This will be replaced with real services (player, library, etc.) in later stories.
type GreetService struct{}

func (g *GreetService) Greet(name string) string {
	return "Hello " + name + ", welcome to Forte!"
}
