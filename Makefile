.PHONY: help
help:
	@echo 'Usage: make [target] ...'
	@echo ''
	@echo 'Targets:'
	@fgrep -h '#!' $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e "s/:.*#!/:/" | column -t -s":"
	@grep '^.PHONY: .* #' Makefile | sed 's/\.PHONY: \(.*\) # \(.*\)/\1:\t\2/' | column -t -s":"

.PHONY: clean # remove dist outfile
clean:
	rm -rf ${OUTFILE}

PHONY: tidy # go mod tidy
tidy:
	go mod tidy

.PHONY: check_fmt # gofmt & test
check_fmt:
	./scripts/check_fmt.sh

.PHONY: test # unit test
test:
	@./scripts/unit_test.sh

.PHONY: show_test_coverage # unit test coverage
show_test_coverage:
	@./scripts/show_coverage.sh

.PHONY: sqlc # run sqlc generate
sqlc:
	sqlc generate

.PHONY: mockery # run mockery
mockery:
	mockery

.PHONY: check # clean tidy check_fmt test workflowcheck
check: clean tidy check_fmt test workflowcheck

.PHONY: openapi_http # generate OpenAPI bindings
openapi_http:
	@./scripts/openapi-http.sh auth internal/auth/ports ports
	@./scripts/openapi-http.sh sales internal/sales/ports ports
	@./scripts/openapi-http.sh servicing internal/servicing/ports/http/dce dce
	@./scripts/openapi-http.sh backoffice internal/servicing/ports/http/backoffice backoffice

.PHONY: temporal # start temporal server
temporal:
	@./scripts/temporal.sh

.PHONY: temporal-send # start a test temporal workflow
temporal-send:
	temporal workflow start --type InstallSoloSaverPlan --run-timeout 600 --task-queue default-task-queue -i '3'

.PHONY: tools # generate tools
tools:
	@./scripts/tools.sh

.PHONY: workflowcheck # run temporal workflow check
workflowcheck:
	workflowcheck -config workflowcheck.config.yaml -show-pos ./...
