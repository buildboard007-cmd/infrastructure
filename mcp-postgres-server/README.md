# MCP PostgreSQL Construction Server

An MCP (Model Context Protocol) server that provides tools to interact with your AWS PostgreSQL construction management database.

## Installation

1. Install dependencies:
```bash
npm install
```

2. Build the project:
```bash
npm run build
```

## Configuration

Set up your environment variables for AWS RDS connection:

```bash
export DATABASE_HOST=your-rds-endpoint.amazonaws.com
export DATABASE_PORT=5432
export DATABASE_NAME=appdb
export DATABASE_USER=postgres
export DATABASE_PASSWORD=your-password
export DATABASE_SSL=true
```

## Claude Desktop Configuration

Add this to your Claude Desktop configuration file:

### macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
### Windows: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "mcp-postgres-construction": {
      "command": "node",
      "args": ["/path/to/mcp-postgres-server/dist/index.js"],
      "env": {
        "DATABASE_HOST": "your-rds-endpoint.amazonaws.com",
        "DATABASE_PORT": "5432",
        "DATABASE_NAME": "appdb",
        "DATABASE_USER": "postgres",
        "DATABASE_PASSWORD": "your-password",
        "DATABASE_SSL": "true"
      }
    }
  }
}
```

## Available Tools

### Core Tools
- **query_database**: Execute any SQL query (SELECT, INSERT, UPDATE, DELETE) - most flexible tool
- **get_database_schema**: Get database schema information for development

### Construction Management Tools
- **get_project_rfis**: Get RFIs for a specific project with user details
- **get_organization_overview**: Get organization statistics and recent projects
- **search_users**: Search users by name, email, or role across organization
- **get_project_details**: Get detailed project info including team and RFI summary

## Usage

After configuration, restart Claude Desktop. The MCP server will be available for use with construction management queries.

Example queries you can ask Claude:
- "Get all users from iam.users with their access contexts using the new RBAC system"
- "Show me project details for project 15"
- "Search for users with email containing 'buildboard'"
- "Get RFIs for project 6"
- "Show me organization overview for org 2"

## Development

For development with hot reload:
```bash
npm run dev
```