> **⚠️ Project Status:** This project is **still in progress**.  
> Features, APIs, and configurations may change frequently until the initial stable release.


# payments-service

A backend service for managing ACH payments using Stripe and Plaid.

## Overview

This service enables secure, reliable, and extensible ACH payment processing. It provides:

- 🔗 **Bank account linking** via Plaid
- 💳 **Customer and payment method creation** in Stripe
- ⚙️ **Orchestration of payment workflows** using Temporal
- 📡 **Webhook handling** for payment updates
- 🗃️ **Persistent tracking** of payments and customer metadata

## Tech Stack

- **Go** – core backend logic
- **Stripe & Plaid APIs** – payment processing and bank account linking
- **Temporal** – long-running, fault-tolerant workflow orchestration
- **PostgreSQL** – persistent storage via `sqlc`
- **Chi** – lightweight HTTP router

## Project Structure

```
/handlers - HTTP handlers
/services - Business logic for Plaid, Stripe, and workflows
/workflow - Temporal workflow and activities
/storage - Postgres + sqlc-based repository layer
/domain - Domain models and interfaces
/sql - Schema and query definitions
```

## Development

1. Copy `.env.example` to `.env` and set your credentials.
2. Run database migrations.
3. Start the Temporal worker.
4. Launch the HTTP server.

## License

MIT © [GalaDe](https://github.com/GalaDe)
