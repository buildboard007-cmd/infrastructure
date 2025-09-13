import { PoolClient, QueryResult } from 'pg';
export interface DatabaseConfig {
    host: string;
    port: number;
    database: string;
    user: string;
    password: string;
    ssl?: any;
}
export declare class DatabaseConnection {
    private pool;
    private isConnected;
    connect(config: DatabaseConfig): Promise<void>;
    disconnect(): Promise<void>;
    query(text: string, params?: any[]): Promise<QueryResult>;
    getClient(): Promise<PoolClient>;
    transaction<T>(callback: (client: PoolClient) => Promise<T>): Promise<T>;
    getProjectRFIs(projectId: string, limit?: number): Promise<any[]>;
}
//# sourceMappingURL=database.d.ts.map