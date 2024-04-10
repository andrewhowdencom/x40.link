# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.9. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for task variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(TASK)
#	@echo "Running task"
#	@$(TASK) <flags/args..>
#
TASK := $(GOBIN)/task-v3.36.0
$(TASK): $(BINGO_DIR)/task.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/task-v3.36.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=task.mod -o=$(GOBIN)/task-v3.36.0 "github.com/go-task/task/v3/cmd/task"

