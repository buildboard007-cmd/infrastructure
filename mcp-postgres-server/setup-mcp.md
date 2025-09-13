# MCP PostgreSQL Setup Instructions

## Database Connection Information
- **Host**: appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com
- **Port**: 5432
- **Database**: appdb
- **Username**: appdb_admin
- **SSL**: Required (true)

## Setup Steps

1. **Get your database password** (choose one method):
   - From AWS Console: Go to RDS → appdb instance → Configuration → Master password
   - From existing Lambda environment variables if configured
   - From AWS Secrets Manager if configured

2. **Configure Claude Desktop**:
   - Open Claude Desktop configuration file:
     - macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
     - Windows: `%APPDATA%\Claude\claude_desktop_config.json`
   
3. **Add the MCP server configuration**:
   ```json
   {
     "mcpServers": {
       "postgres-construction": {
         "command": "node",
         "args": ["/Users/mayur/git_personal/infrastructure/mcp-postgres-server/dist/index.js"],
         "env": {
           "DATABASE_HOST": "appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com",
           "DATABASE_PORT": "5432",
           "DATABASE_NAME": "appdb",
           "DATABASE_USER": "appdb_admin",
           "DATABASE_PASSWORD": "YOUR_ACTUAL_PASSWORD",
           "DATABASE_SSL": "true"
         }
       }
     }
   }
   ```

4. **Replace YOUR_ACTUAL_PASSWORD** with your database password

5. **Restart Claude Desktop** for changes to take effect

## Testing the Connection

After restarting Claude Desktop, test with these queries:
- "Show me the database schema"
- "List all tables in the iam schema"
- "Get user with email example@email.com"

## Available Operations

With read/write permissions enabled, you can:
- **SELECT**: Query any data
- **INSERT**: Add new records
- **UPDATE**: Modify existing records
- **DELETE**: Remove records

Example queries:
```sql
-- Get all organizations
SELECT * FROM iam.organizations;

-- Update user status
UPDATE iam.users SET status = 'active' WHERE email = 'user@example.com';

-- Insert new location
INSERT INTO iam.locations (name, org_id, location_type) VALUES ('New Site', 1, 'construction_site');
```

## Security Note
Keep your database password secure. Never commit the configuration file with the password to version control.