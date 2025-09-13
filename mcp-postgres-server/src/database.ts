import { Pool, PoolClient, QueryResult } from 'pg';

export interface DatabaseConfig {
  host: string;
  port: number;
  database: string;
  user: string;
  password: string;
  ssl?: any;
}

export class DatabaseConnection {
  private pool: Pool | null = null;
  private isConnected = false;

  async connect(config: DatabaseConfig): Promise<void> {
    if (this.isConnected) {
      return;
    }

    this.pool = new Pool({
      ...config,
      max: 10, // Maximum number of clients in the pool
      idleTimeoutMillis: 30000, // Close idle clients after 30 seconds
      connectionTimeoutMillis: 10000, // Return an error after 10 seconds if connection could not be established
    });

    // Test the connection
    try {
      const client = await this.pool.connect();
      await client.query('SELECT NOW()');
      client.release();
      this.isConnected = true;
      console.error('Database connection established successfully');
    } catch (error) {
      console.error('Failed to connect to database:', error);
      throw error;
    }
  }

  async disconnect(): Promise<void> {
    if (this.pool) {
      await this.pool.end();
      this.pool = null;
      this.isConnected = false;
      console.error('Database connection closed');
    }
  }

  async query(text: string, params?: any[]): Promise<QueryResult> {
    if (!this.pool) {
      throw new Error('Database not connected');
    }

    try {
      const start = Date.now();
      const result = await this.pool.query(text, params);
      const duration = Date.now() - start;
      
      console.error(`Query executed in ${duration}ms: ${text.substring(0, 100)}${text.length > 100 ? '...' : ''}`);
      
      return result;
    } catch (error) {
      console.error('Database query error:', error);
      console.error('Query:', text);
      console.error('Params:', params);
      throw error;
    }
  }

  async getClient(): Promise<PoolClient> {
    if (!this.pool) {
      throw new Error('Database not connected');
    }
    return this.pool.connect();
  }

  // Helper method for transactions
  async transaction<T>(callback: (client: PoolClient) => Promise<T>): Promise<T> {
    const client = await this.getClient();
    
    try {
      await client.query('BEGIN');
      const result = await callback(client);
      await client.query('COMMIT');
      return result;
    } catch (error) {
      await client.query('ROLLBACK');
      throw error;
    } finally {
      client.release();
    }
  }

  // Construction management specific helper queries
  async getProjectRFIs(projectId: string, limit = 50): Promise<any[]> {
    const query = `
      SELECT 
        r.id, r.rfi_number, r.title, r.description, r.priority, r.status,
        r.submitted_date, r.due_date, r.response_date,
        u1.first_name || ' ' || u1.last_name as submitted_by_name,
        u2.first_name || ' ' || u2.last_name as assigned_to_name
      FROM project.rfis r
      LEFT JOIN iam.users u1 ON r.submitted_by = u1.id
      LEFT JOIN iam.users u2 ON r.assigned_to = u2.id
      WHERE r.project_id = $1 AND r.is_deleted = false
      ORDER BY r.created_at DESC
      LIMIT $2
    `;
    
    const result = await this.query(query, [projectId, limit]);
    return result.rows;
  }
}