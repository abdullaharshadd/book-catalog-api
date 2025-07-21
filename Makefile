# Makefile

.PHONY: help install dev-install test test-cov lint format clean run docker-build docker-run

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install production dependencies
	pip install -r requirements.txt

dev-install: ## Install all dependencies including development ones
	pip install -r requirements.txt
	pip install -e ".[test,dev]"

test: ## Run tests
	pytest -v

test-cov: ## Run tests with coverage
	pytest --cov=app --cov-report=html --cov-report=term -v

lint: ## Run linters
	flake8 app tests
	black --check app tests
	isort --check-only app tests

format: ## Format code
	black app tests
	isort app tests

clean: ## Clean up generated files
	find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete
	rm -rf htmlcov/
	rm -rf .coverage
	rm -rf .pytest_cache/
	rm -rf dist/
	rm -rf build/
	rm -rf *.egg-info/
	rm -f books.db

run: ## Run the development server
	uvicorn app.main:app --reload --host 0.0.0.0 --port 8000

run-prod: ## Run with Gunicorn (production)
	gunicorn app.main:app -w 4 -k uvicorn.workers.UvicornWorker --bind 0.0.0.0:8000

docker-build: ## Build Docker image
	docker build -t book-catalog-api .

docker-run: ## Run Docker container
	docker run -p 8000:8000 book-catalog-api

docker-compose-up: ## Run with docker-compose
	docker-compose up --build

docker-compose-down: ## Stop docker-compose
	docker-compose down

setup-dev: ## Set up development environment
	python -m venv venv
	@echo "Activate the virtual environment with: source venv/bin/activate (Linux/Mac) or venv\\Scripts\\activate (Windows)"
	@echo "Then run: make dev-install"

check: lint test ## Run all checks (lint + test)

all: clean format check ## Clean, format, and test everything