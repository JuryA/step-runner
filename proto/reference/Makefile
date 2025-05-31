.PHONY: build test clean

build:
	go build -o step-runner main.go

test: build
	./step-runner ./examples/hello_world.textproto
	./step-runner ./examples/composition.textproto

clean:
	rm -f step-runner