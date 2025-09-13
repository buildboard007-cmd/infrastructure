#!/usr/bin/env node

/**
 * MCP Server for AWS PostgreSQL Construction Management Database
 * 
 * This server provides tools to interact with a construction management database
 * including users, organizations, locations, projects, RFIs, and role management.
 */

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  Tool,
} from '@modelcontextprotocol/sdk/types.js';
import { z } from 'zod';
import { DatabaseConnection } from './database.js';
import { ConstructionTools } from './tools.js';

// Server configuration
const SERVER_NAME = 'mcp-postgres-construction-server';
const SERVER_VERSION = '1.0.0';

// Environment configuration schema
const ConfigSchema = z.object({
  DATABASE_HOST: z.string().default('localhost'),
  DATABASE_PORT: z.string().default('5432'),
  DATABASE_NAME: z.string().default('appdb'),
  DATABASE_USER: z.string().default('postgres'),
  DATABASE_PASSWORD: z.string(),
  DATABASE_SSL: z.string().default('false'),
});

class ConstructionMCPServer {
  private server: Server;
  private db: DatabaseConnection;
  private tools: ConstructionTools;

  constructor() {
    this.server = new Server({
      name: SERVER_NAME,
      version: SERVER_VERSION,
    });

    // Initialize database connection
    this.db = new DatabaseConnection();
    this.tools = new ConstructionTools(this.db);

    this.setupToolHandlers();
  }

  private setupToolHandlers(): void {
    // List available tools
    this.server.setRequestHandler(ListToolsRequestSchema, async () => {
      return {
        tools: this.tools.getToolDefinitions(),
      };
    });

    // Handle tool calls
    this.server.setRequestHandler(CallToolRequestSchema, async (request) => {
      const { name, arguments: args } = request.params;

      try {
        const result = await this.tools.executeTool(name, args || {});
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(result, null, 2),
            },
          ],
        };
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Unknown error';
        return {
          content: [
            {
              type: 'text',
              text: `Error executing tool ${name}: ${errorMessage}`,
            },
          ],
          isError: true,
        };
      }
    });
  }

  async start(): Promise<void> {
    // Parse environment configuration
    const config = ConfigSchema.parse(process.env);
    
    // Connect to database
    await this.db.connect({
      host: config.DATABASE_HOST,
      port: parseInt(config.DATABASE_PORT),
      database: config.DATABASE_NAME,
      user: config.DATABASE_USER,
      password: config.DATABASE_PASSWORD,
      ssl: config.DATABASE_SSL === 'true' ? { rejectUnauthorized: false } : false,
    });

    // Start the server
    const transport = new StdioServerTransport();
    await this.server.connect(transport);
    
    console.error(`${SERVER_NAME} v${SERVER_VERSION} connected to PostgreSQL at ${config.DATABASE_HOST}:${config.DATABASE_PORT}/${config.DATABASE_NAME}`);
  }

  async stop(): Promise<void> {
    await this.db.disconnect();
    await this.server.close();
  }
}

// Main execution
async function main(): Promise<void> {
  const server = new ConstructionMCPServer();
  
  // Handle graceful shutdown
  process.on('SIGINT', async () => {
    console.error('Received SIGINT, shutting down gracefully...');
    await server.stop();
    process.exit(0);
  });

  process.on('SIGTERM', async () => {
    console.error('Received SIGTERM, shutting down gracefully...');
    await server.stop();
    process.exit(0);
  });

  try {
    await server.start();
  } catch (error) {
    console.error('Failed to start server:', error);
    process.exit(1);
  }
}

// Run the server
main().catch((error) => {
  console.error('Unhandled error:', error);
  process.exit(1);
});