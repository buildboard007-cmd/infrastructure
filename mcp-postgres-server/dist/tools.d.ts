import { Tool } from '@modelcontextprotocol/sdk/types.js';
import { DatabaseConnection } from './database.js';
export declare class ConstructionTools {
    private db;
    constructor(db: DatabaseConnection);
    getToolDefinitions(): Tool[];
    executeTool(name: string, args: any): Promise<any>;
    private queryDatabase;
    private getProjectRFIs;
    private getOrganizationOverview;
    private searchUsers;
    private getProjectDetails;
    private getDatabaseSchema;
}
//# sourceMappingURL=tools.d.ts.map