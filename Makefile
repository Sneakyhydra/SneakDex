# File: Makefile

# Variables
DC_DEV=docker-compose -f docker-compose.dev.yml
DC_PROD=docker-compose -f docker-compose.yml

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
PURPLE=\033[0;35m
CYAN=\033[0;36m
WHITE=\033[1;37m
NC=\033[0m # No Color

# General Commands
.PHONY: help \
        up-dev down-dev start-dev stop-dev restart-dev build-dev rebuild-dev logs-dev status-dev reset-dev clean-dev \
        up-prod down-prod start-prod stop-prod restart-prod build-prod rebuild-prod logs-prod status-prod reset-prod clean-prod \
        start-crawler-dev stop-crawler-dev restart-crawler-dev logs-crawler-dev \
        start-parser-dev stop-parser-dev restart-parser-dev logs-parser-dev \
        start-indexer-dev stop-indexer-dev restart-indexer-dev logs-indexer-dev \
        start-frontend-dev stop-frontend-dev restart-frontend-dev logs-frontend-dev \
        start-query-api-dev stop-query-api-dev restart-query-api-dev logs-query-api-dev \
        start-kafka stop-kafka restart-kafka logs-kafka \
        start-redis stop-redis restart-redis logs-redis \
        exec-crawler-dev exec-parser-dev exec-indexer-dev exec-frontend-dev exec-query-api-dev exec-kafka exec-redis \
        kafka-topics kafka-create-topics kafka-list-topics kafka-delete-topics \
        install-deps update-deps

# Help
help:
	@echo "$(CYAN)╔════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(CYAN)║                    SneakDex Makefile Commands                   ║$(NC)"
	@echo "$(CYAN)╚════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(GREEN)🛠  DEVELOPMENT ENVIRONMENT$(NC)"
	@echo "  $(YELLOW)make up-dev$(NC)           Start all dev services (with build)"
	@echo "  $(YELLOW)make start-dev$(NC)        Start dev services (no build)"
	@echo "  $(YELLOW)make stop-dev$(NC)         Stop dev services (keep containers)"
	@echo "  $(YELLOW)make restart-dev$(NC)      Restart dev services"
	@echo "  $(YELLOW)make down-dev$(NC)         Stop and remove dev containers"
	@echo "  $(YELLOW)make build-dev$(NC)        Build dev images"
	@echo "  $(YELLOW)make rebuild-dev$(NC)      Force rebuild dev images"
	@echo "  $(YELLOW)make logs-dev$(NC)         View all dev logs"
	@echo "  $(YELLOW)make status-dev$(NC)       Show dev services status"
	@echo "  $(YELLOW)make reset-dev$(NC)        Full reset of dev environment"
	@echo "  $(YELLOW)make clean-dev$(NC)        Clean dev containers and volumes"
	@echo ""
	@echo "$(GREEN)🚀 PRODUCTION ENVIRONMENT$(NC)"
	@echo "  $(BLUE)make up-prod$(NC)          Start all prod services"
	@echo "  $(BLUE)make start-prod$(NC)       Start prod services (no build)"
	@echo "  $(BLUE)make stop-prod$(NC)        Stop prod services"
	@echo "  $(BLUE)make restart-prod$(NC)     Restart prod services"
	@echo "  $(BLUE)make down-prod$(NC)        Stop and remove prod containers"
	@echo "  $(BLUE)make build-prod$(NC)       Build prod images"
	@echo "  $(BLUE)make rebuild-prod$(NC)     Force rebuild prod images"
	@echo "  $(BLUE)make logs-prod$(NC)        View all prod logs"
	@echo "  $(BLUE)make status-prod$(NC)      Show prod services status"
	@echo "  $(BLUE)make reset-prod$(NC)       Full reset of prod environment"
	@echo "  $(BLUE)make clean-prod$(NC)       Clean prod containers and volumes"
	@echo ""
	@echo "$(GREEN)🔧 INDIVIDUAL SERVICE MANAGEMENT (DEV)$(NC)"
	@echo "  $(PURPLE)make start-crawler-dev$(NC)     Start crawler service"
	@echo "  $(PURPLE)make stop-crawler-dev$(NC)      Stop crawler service"
	@echo "  $(PURPLE)make restart-crawler-dev$(NC)   Restart crawler service"
	@echo "  $(PURPLE)make logs-crawler-dev$(NC)      View crawler logs"
	@echo "  $(PURPLE)make start-parser-dev$(NC)      Start parser service"
	@echo "  $(PURPLE)make stop-parser-dev$(NC)       Stop parser service"
	@echo "  $(PURPLE)make restart-parser-dev$(NC)    Restart parser service"
	@echo "  $(PURPLE)make logs-parser-dev$(NC)       View parser logs"
	@echo "  $(PURPLE)make start-indexer-dev$(NC)     Start indexer service"
	@echo "  $(PURPLE)make stop-indexer-dev$(NC)      Stop indexer service"
	@echo "  $(PURPLE)make restart-indexer-dev$(NC)   Restart indexer service"
	@echo "  $(PURPLE)make logs-indexer-dev$(NC)      View indexer logs"
	@echo "  $(PURPLE)make start-frontend-dev$(NC)    Start frontend service"
	@echo "  $(PURPLE)make stop-frontend-dev$(NC)     Stop frontend service"
	@echo "  $(PURPLE)make restart-frontend-dev$(NC)  Restart frontend service"
	@echo "  $(PURPLE)make logs-frontend-dev$(NC)     View frontend logs"
	@echo "  $(PURPLE)make start-query-api-dev$(NC)   Start query-api service"
	@echo "  $(PURPLE)make stop-query-api-dev$(NC)    Stop query-api service"
	@echo "  $(PURPLE)make restart-query-api-dev$(NC) Restart query-api service"
	@echo "  $(PURPLE)make logs-query-api-dev$(NC)    View query-api logs"
	@echo ""
	@echo "$(GREEN)🗃️  INFRASTRUCTURE SERVICES$(NC)"
	@echo "  $(CYAN)make start-kafka$(NC)       Start Kafka service"
	@echo "  $(CYAN)make stop-kafka$(NC)        Stop Kafka service"
	@echo "  $(CYAN)make restart-kafka$(NC)     Restart Kafka service"
	@echo "  $(CYAN)make logs-kafka$(NC)        View Kafka logs"
	@echo "  $(CYAN)make start-redis$(NC)       Start Redis service"
	@echo "  $(CYAN)make stop-redis$(NC)        Stop Redis service"
	@echo "  $(CYAN)make restart-redis$(NC)     Restart Redis service"
	@echo "  $(CYAN)make logs-redis$(NC)        View Redis logs"
	@echo ""
	@echo "$(GREEN)🐚 SHELL ACCESS$(NC)"
	@echo "  $(WHITE)make exec-crawler-dev$(NC)   Shell into crawler container"
	@echo "  $(WHITE)make exec-parser-dev$(NC)    Shell into parser container"
	@echo "  $(WHITE)make exec-indexer-dev$(NC)   Shell into indexer container"
	@echo "  $(WHITE)make exec-frontend-dev$(NC)  Shell into frontend container"
	@echo "  $(WHITE)make exec-query-api-dev$(NC) Shell into query-api container"
	@echo "  $(WHITE)make exec-kafka$(NC)         Shell into Kafka container"
	@echo "  $(WHITE)make exec-redis$(NC)         Shell into Redis container"
	@echo ""
	@echo "$(GREEN)📊 KAFKA MANAGEMENT$(NC)"
	@echo "  $(YELLOW)make kafka-topics$(NC)       List all Kafka topics"
	@echo "  $(YELLOW)make kafka-create-topics$(NC) Create default topics"
	@echo "  $(YELLOW)make kafka-list-topics$(NC)  List topics with details"
	@echo "  $(YELLOW)make kafka-delete-topics$(NC) Delete all topics"
	@echo ""
	@echo "$(GREEN)📦 DEPENDENCIES$(NC)"
	@echo "  $(PURPLE)make install-deps$(NC)      Install/update dependencies"
	@echo "  $(PURPLE)make update-deps$(NC)       Update all dependencies"

# ============================================================================
# DEVELOPMENT ENVIRONMENT
# ============================================================================

up-dev:
	@echo "$(GREEN)🚀 Starting development environment...$(NC)"
	$(DC_DEV) up --build -d

start-dev:
	@echo "$(GREEN)▶️  Starting development services (no build)...$(NC)"
	$(DC_DEV) up -d

stop-dev:
	@echo "$(YELLOW)⏹️  Stopping development services...$(NC)"
	$(DC_DEV) stop

restart-dev:
	@echo "$(BLUE)🔄 Restarting development services...$(NC)"
	$(DC_DEV) restart

down-dev:
	@echo "$(RED)🛑 Stopping and removing development containers...$(NC)"
	$(DC_DEV) down

build-dev:
	@echo "$(BLUE)🔨 Building development images...$(NC)"
	$(DC_DEV) build

rebuild-dev:
	@echo "$(BLUE)🔨 Force rebuilding development images...$(NC)"
	$(DC_DEV) build --no-cache

logs-dev:
	@echo "$(CYAN)📋 Viewing development logs...$(NC)"
	$(DC_DEV) logs -f --tail=100

status-dev:
	@echo "$(CYAN)📊 Development services status:$(NC)"
	$(DC_DEV) ps

reset-dev:
	@echo "$(RED)🔄 Resetting development environment...$(NC)"
	$(DC_DEV) down -v
	$(DC_DEV) up --build -d

clean-dev:
	@echo "$(RED)🧹 Cleaning development environment...$(NC)"
	$(DC_DEV) down -v --remove-orphans
	docker system prune -f

# ============================================================================
# PRODUCTION ENVIRONMENT
# ============================================================================

up-prod:
	@echo "$(GREEN)🚀 Starting production environment...$(NC)"
	$(DC_PROD) up --build -d

start-prod:
	@echo "$(GREEN)▶️  Starting production services (no build)...$(NC)"
	$(DC_PROD) up -d

stop-prod:
	@echo "$(YELLOW)⏹️  Stopping production services...$(NC)"
	$(DC_PROD) stop

restart-prod:
	@echo "$(BLUE)🔄 Restarting production services...$(NC)"
	$(DC_PROD) restart

down-prod:
	@echo "$(RED)🛑 Stopping and removing production containers...$(NC)"
	$(DC_PROD) down

build-prod:
	@echo "$(BLUE)🔨 Building production images...$(NC)"
	$(DC_PROD) build

rebuild-prod:
	@echo "$(BLUE)🔨 Force rebuilding production images...$(NC)"
	$(DC_PROD) build --no-cache

logs-prod:
	@echo "$(CYAN)📋 Viewing production logs...$(NC)"
	$(DC_PROD) logs -f --tail=100

status-prod:
	@echo "$(CYAN)📊 Production services status:$(NC)"
	$(DC_PROD) ps

reset-prod:
	@echo "$(RED)🔄 Resetting production environment...$(NC)"
	$(DC_PROD) down -v
	$(DC_PROD) up --build -d

clean-prod:
	@echo "$(RED)🧹 Cleaning production environment...$(NC)"
	$(DC_PROD) down -v --remove-orphans
	docker system prune -f

# ============================================================================
# INDIVIDUAL SERVICE MANAGEMENT (DEV)
# ============================================================================

# Crawler Service
start-crawler-dev:
	@echo "$(PURPLE)🕷️  Starting crawler service...$(NC)"
	$(DC_DEV) up -d crawler-dev

stop-crawler-dev:
	@echo "$(PURPLE)⏹️  Stopping crawler service...$(NC)"
	$(DC_DEV) stop crawler-dev

restart-crawler-dev:
	@echo "$(PURPLE)🔄 Restarting crawler service...$(NC)"
	$(DC_DEV) restart crawler-dev

logs-crawler-dev:
	@echo "$(PURPLE)📋 Viewing crawler logs...$(NC)"
	$(DC_DEV) logs -f crawler-dev

# Parser Service
start-parser-dev:
	@echo "$(PURPLE)🔍 Starting parser service...$(NC)"
	$(DC_DEV) up -d parser-dev

stop-parser-dev:
	@echo "$(PURPLE)⏹️  Stopping parser service...$(NC)"
	$(DC_DEV) stop parser-dev

restart-parser-dev:
	@echo "$(PURPLE)🔄 Restarting parser service...$(NC)"
	$(DC_DEV) restart parser-dev

logs-parser-dev:
	@echo "$(PURPLE)📋 Viewing parser logs...$(NC)"
	$(DC_DEV) logs -f parser-dev

# Indexer Service
start-indexer-dev:
	@echo "$(PURPLE)🗂️  Starting indexer service...$(NC)"
	$(DC_DEV) up -d indexer-dev

stop-indexer-dev:
	@echo "$(PURPLE)⏹️  Stopping indexer service...$(NC)"
	$(DC_DEV) stop indexer-dev

restart-indexer-dev:
	@echo "$(PURPLE)🔄 Restarting indexer service...$(NC)"
	$(DC_DEV) restart indexer-dev

logs-indexer-dev:
	@echo "$(PURPLE)📋 Viewing indexer logs...$(NC)"
	$(DC_DEV) logs -f indexer-dev

# Frontend Service
start-frontend-dev:
	@echo "$(PURPLE)🌐 Starting frontend service...$(NC)"
	$(DC_DEV) up -d frontend-dev

stop-frontend-dev:
	@echo "$(PURPLE)⏹️  Stopping frontend service...$(NC)"
	$(DC_DEV) stop frontend-dev

restart-frontend-dev:
	@echo "$(PURPLE)🔄 Restarting frontend service...$(NC)"
	$(DC_DEV) restart frontend-dev

logs-frontend-dev:
	@echo "$(PURPLE)📋 Viewing frontend logs...$(NC)"
	$(DC_DEV) logs -f frontend-dev

# Query API Service
start-query-api-dev:
	@echo "$(PURPLE)🔌 Starting query-api service...$(NC)"
	$(DC_DEV) up -d query-api-dev

stop-query-api-dev:
	@echo "$(PURPLE)⏹️  Stopping query-api service...$(NC)"
	$(DC_DEV) stop query-api-dev

restart-query-api-dev:
	@echo "$(PURPLE)🔄 Restarting query-api service...$(NC)"
	$(DC_DEV) restart query-api-dev

logs-query-api-dev:
	@echo "$(PURPLE)📋 Viewing query-api logs...$(NC)"
	$(DC_DEV) logs -f query-api-dev

# ============================================================================
# INFRASTRUCTURE SERVICES
# ============================================================================

# Kafka
start-kafka:
	@echo "$(CYAN)🎯 Starting Kafka service...$(NC)"
	$(DC_DEV) up -d kafka zookeeper

stop-kafka:
	@echo "$(CYAN)⏹️  Stopping Kafka service...$(NC)"
	$(DC_DEV) stop kafka zookeeper

restart-kafka:
	@echo "$(CYAN)🔄 Restarting Kafka service...$(NC)"
	$(DC_DEV) restart kafka zookeeper

logs-kafka:
	@echo "$(CYAN)📋 Viewing Kafka logs...$(NC)"
	$(DC_DEV) logs -f kafka

# Redis
start-redis:
	@echo "$(CYAN)🗄️  Starting Redis service...$(NC)"
	$(DC_DEV) up -d redis

stop-redis:
	@echo "$(CYAN)⏹️  Stopping Redis service...$(NC)"
	$(DC_DEV) stop redis

restart-redis:
	@echo "$(CYAN)🔄 Restarting Redis service...$(NC)"
	$(DC_DEV) restart redis

logs-redis:
	@echo "$(CYAN)📋 Viewing Redis logs...$(NC)"
	$(DC_DEV) logs -f redis

# ============================================================================
# SHELL ACCESS
# ============================================================================

exec-crawler-dev:
	@echo "$(WHITE)🐚 Entering crawler container...$(NC)"
	$(DC_DEV) exec crawler-dev sh

exec-parser-dev:
	@echo "$(WHITE)🐚 Entering parser container...$(NC)"
	$(DC_DEV) exec parser-dev sh

exec-indexer-dev:
	@echo "$(WHITE)🐚 Entering indexer container...$(NC)"
	$(DC_DEV) exec indexer-dev sh

exec-frontend-dev:
	@echo "$(WHITE)🐚 Entering frontend container...$(NC)"
	$(DC_DEV) exec frontend-dev sh

exec-query-api-dev:
	@echo "$(WHITE)🐚 Entering query-api container...$(NC)"
	$(DC_DEV) exec query-api-dev sh

exec-kafka:
	@echo "$(WHITE)🐚 Entering Kafka container...$(NC)"
	$(DC_DEV) exec kafka sh

exec-redis:
	@echo "$(WHITE)🐚 Entering Redis container...$(NC)"
	$(DC_DEV) exec redis sh

# ============================================================================
# KAFKA MANAGEMENT
# ============================================================================

kafka-topics:
	@echo "$(YELLOW)📊 Listing Kafka topics...$(NC)"
	$(DC_DEV) exec kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-create-topics:
	@echo "$(YELLOW)➕ Creating Kafka topics...$(NC)"
	$(DC_DEV) exec kafka kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic raw-html --partitions 3 --replication-factor 1
	$(DC_DEV) exec kafka kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic parsed-pages --partitions 3 --replication-factor 1
	$(DC_DEV) exec kafka kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic indexed-pages --partitions 3 --replication-factor 1
	@echo "$(GREEN)✅ Topics created successfully!$(NC)"

kafka-list-topics:
	@echo "$(YELLOW)📋 Kafka topics details...$(NC)"
	$(DC_DEV) exec kafka kafka-topics --bootstrap-server localhost:9092 --describe

kafka-delete-topics:
	@echo "$(RED)🗑️  Deleting Kafka topics...$(NC)"
	$(DC_DEV) exec kafka kafka-topics --bootstrap-server localhost:9092 --delete --topic raw-html || true
	$(DC_DEV) exec kafka kafka-topics --bootstrap-server localhost:9092 --delete --topic parsed-pages || true
	$(DC_DEV) exec kafka kafka-topics --bootstrap-server localhost:9092 --delete --topic indexed-pages || true

# ============================================================================
# DEPENDENCIES
# ============================================================================

install-deps:
	@echo "$(PURPLE)📦 Installing dependencies...$(NC)"
	$(DC_DEV) exec crawler-dev go mod tidy || true
	$(DC_DEV) exec parser-dev pip install -r requirements.txt || true
	$(DC_DEV) exec indexer-dev pip install -r requirements.txt || true
	$(DC_DEV) exec frontend-dev npm install || true

update-deps:
	@echo "$(PURPLE)⬆️  Updating dependencies...$(NC)"
	$(DC_DEV) exec crawler-dev go get -u ./... || true
	$(DC_DEV) exec parser-dev pip install --upgrade -r requirements.txt || true
	$(DC_DEV) exec indexer-dev pip install --upgrade -r requirements.txt || true
	$(DC_DEV) exec frontend-dev npm update || true