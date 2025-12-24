test: 
	@echo "Запуск тестов"
	@go test -v -cover ./...

lint:
	@echo "Запуск проверок линтера"
	@golangci-lint run ./... 