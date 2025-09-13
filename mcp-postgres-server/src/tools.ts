import { Tool } from '@modelcontextprotocol/sdk/types.js';
import { z } from 'zod';
import { DatabaseConnection } from './database.js';

export class ConstructionTools {
  constructor(private db: DatabaseConnection) {}

  getToolDefinitions(): Tool[] {
    return [
      {
        name: 'query_database',
        description: 'Execute a SQL query against the construction management database',
        inputSchema: {
          type: 'object',
          properties: {
            query: {
              type: 'string',
              description: 'SQL query to execute (supports SELECT, INSERT, UPDATE, DELETE)',
            },
            params: {
              type: 'array',
              items: { type: 'string' },
              description: 'Query parameters for prepared statements',
            },
          },
          required: ['query'],
        },
      },
      {
        name: 'get_project_rfis',
        description: 'Get RFIs (Request for Information) for a specific project',
        inputSchema: {
          type: 'object',
          properties: {
            projectId: {
              type: 'string',
              description: 'Project ID',
            },
            limit: {
              type: 'number',
              description: 'Maximum number of RFIs to return (default: 50)',
              default: 50,
            },
          },
          required: ['projectId'],
        },
      },
      {
        name: 'get_organization_overview',
        description: 'Get organization overview including users, locations, and projects',
        inputSchema: {
          type: 'object',
          properties: {
            orgId: {
              type: 'string',
              description: 'Organization ID',
            },
          },
          required: ['orgId'],
        },
      },
      {
        name: 'search_users',
        description: 'Search users by name, email, or role',
        inputSchema: {
          type: 'object',
          properties: {
            searchTerm: {
              type: 'string',
              description: 'Search term (name, email, or role)',
            },
            orgId: {
              type: 'string',
              description: 'Organization ID to limit search scope',
            },
            limit: {
              type: 'number',
              description: 'Maximum number of results (default: 20)',
              default: 20,
            },
          },
          required: ['searchTerm'],
        },
      },
      {
        name: 'get_project_details',
        description: 'Get detailed information about a specific project',
        inputSchema: {
          type: 'object',
          properties: {
            projectId: {
              type: 'string',
              description: 'Project ID',
            },
          },
          required: ['projectId'],
        },
      },
      {
        name: 'get_database_schema',
        description: 'Get database schema information for tables and columns',
        inputSchema: {
          type: 'object',
          properties: {
            tableName: {
              type: 'string',
              description: 'Specific table name (optional, returns all tables if not provided)',
            },
            schemaName: {
              type: 'string',
              description: 'Schema name (default: public)',
              default: 'public',
            },
          },
        },
      },
    ];
  }

  async executeTool(name: string, args: any): Promise<any> {
    switch (name) {
      case 'query_database':
        return this.queryDatabase(args);
      case 'get_project_rfis':
        return this.getProjectRFIs(args);
      case 'get_organization_overview':
        return this.getOrganizationOverview(args);
      case 'search_users':
        return this.searchUsers(args);
      case 'get_project_details':
        return this.getProjectDetails(args);
      case 'get_database_schema':
        return this.getDatabaseSchema(args);
      default:
        throw new Error(`Unknown tool: ${name}`);
    }
  }

  private async queryDatabase(args: { query: string; params?: string[] }): Promise<any> {
    const { query, params = [] } = args;

    // Allow both read and write operations
    const result = await this.db.query(query, params);
    return {
      rows: result.rows,
      rowCount: result.rowCount,
      query: query,
    };
  }

  private async getProjectRFIs(args: { projectId: string; limit?: number }): Promise<any> {
    return await this.db.getProjectRFIs(args.projectId, args.limit || 50);
  }

  private async getOrganizationOverview(args: { orgId: string }): Promise<any> {
    const queries = [
      // Organization info
      `SELECT * FROM iam.organizations WHERE id = $1`,
      // User count
      `SELECT COUNT(*) as user_count FROM iam.users WHERE org_id = $1 AND is_deleted = false`,
      // Location count
      `SELECT COUNT(*) as location_count FROM iam.locations WHERE org_id = $1 AND is_deleted = false`,
      // Project count
      `SELECT COUNT(*) as project_count FROM projects p 
       JOIN iam.locations l ON p.location_id = l.id 
       WHERE l.org_id = $1 AND p.is_deleted = false`,
      // Recent projects
      `SELECT p.id, p.name, p.project_stage, l.name as location_name
       FROM projects p 
       JOIN iam.locations l ON p.location_id = l.id
       WHERE l.org_id = $1 AND p.is_deleted = false
       ORDER BY p.created_at DESC LIMIT 5`,
    ];

    const [orgInfo, userCount, locationCount, projectCount, recentProjects] = await Promise.all(
      queries.map(query => this.db.query(query, [args.orgId]))
    );

    return {
      organization: orgInfo.rows[0],
      stats: {
        users: userCount.rows[0].user_count,
        locations: locationCount.rows[0].location_count,
        projects: projectCount.rows[0].project_count,
      },
      recentProjects: recentProjects.rows,
    };
  }

  private async searchUsers(args: { searchTerm: string; orgId?: string; limit?: number }): Promise<any> {
    const { searchTerm, orgId, limit = 20 } = args;
    
    let query = `
      SELECT 
        u.id, u.email, u.first_name, u.last_name, u.job_title, u.status,
        o.name as org_name
      FROM iam.users u
      JOIN iam.organizations o ON u.org_id = o.id
      WHERE u.is_deleted = false
        AND (
          u.first_name ILIKE $1 OR 
          u.last_name ILIKE $1 OR 
          u.email ILIKE $1
        )
    `;
    
    const params = [`%${searchTerm}%`];
    
    if (orgId) {
      query += ` AND u.org_id = $2`;
      params.push(orgId);
    }
    
    query += ` ORDER BY u.first_name, u.last_name LIMIT $${params.length + 1}`;
    params.push(limit.toString());

    const result = await this.db.query(query, params);
    return result.rows;
  }

  private async getProjectDetails(args: { projectId: string }): Promise<any> {
    const queries = [
      // Project info
      `SELECT p.*, l.name as location_name, o.name as org_name
       FROM projects p
       JOIN iam.locations l ON p.location_id = l.id
       JOIN iam.organizations o ON l.org_id = o.id
       WHERE p.id = $1`,
      // Project team
      `SELECT 
         u.id, u.first_name, u.last_name, u.email, u.job_title,
         r.name as role_name, pur.trade_type, pur.is_primary
       FROM iam.project_user_roles pur
       JOIN iam.users u ON pur.user_id = u.id
       JOIN iam.roles r ON pur.role_id = r.id
       WHERE pur.project_id = $1 AND pur.is_deleted = false
       ORDER BY pur.is_primary DESC, r.name`,
      // RFI summary
      `SELECT status, COUNT(*) as count
       FROM project.rfis
       WHERE project_id = $1 AND is_deleted = false
       GROUP BY status`,
    ];

    const [projectInfo, projectTeam, rfiSummary] = await Promise.all(
      queries.map(query => this.db.query(query, [args.projectId]))
    );

    return {
      project: projectInfo.rows[0],
      team: projectTeam.rows,
      rfiSummary: rfiSummary.rows,
    };
  }

  private async getDatabaseSchema(args: { tableName?: string; schemaName?: string }): Promise<any> {
    const { tableName, schemaName = 'public' } = args;
    
    let query = `
      SELECT 
        t.table_schema,
        t.table_name,
        c.column_name,
        c.data_type,
        c.is_nullable,
        c.column_default,
        c.character_maximum_length
      FROM information_schema.tables t
      JOIN information_schema.columns c ON t.table_name = c.table_name 
        AND t.table_schema = c.table_schema
      WHERE t.table_schema IN ('public', 'iam', 'project')
    `;
    
    const params: string[] = [];
    
    if (tableName) {
      query += ` AND t.table_name = $1`;
      params.push(tableName);
    }
    
    query += ` ORDER BY t.table_schema, t.table_name, c.ordinal_position`;

    const result = await this.db.query(query, params);
    
    // Group by table
    const tables: { [key: string]: any } = {};
    result.rows.forEach(row => {
      const tableKey = `${row.table_schema}.${row.table_name}`;
      if (!tables[tableKey]) {
        tables[tableKey] = {
          schema: row.table_schema,
          name: row.table_name,
          columns: [],
        };
      }
      tables[tableKey].columns.push({
        name: row.column_name,
        type: row.data_type,
        nullable: row.is_nullable === 'YES',
        default: row.column_default,
        maxLength: row.character_maximum_length,
      });
    });

    return {
      tables: Object.values(tables),
    };
  }
}