import { Construct } from "constructs";
import * as ec2 from "aws-cdk-lib/aws-ec2";

export class SharedLambdaSecurityGroup extends Construct {
    public readonly securityGroup: ec2.ISecurityGroup;

    constructor(scope: Construct, id: string, vpc: ec2.IVpc) {
        super(scope, id);

        // Create the shared security group for all Lambdas
        // CDK will handle this properly - if it already exists in the stack, it won't duplicate
        // For cross-stack usage, you would need to export/import this security group
        this.securityGroup = new ec2.SecurityGroup(this, "SharedLambdaDatabaseAccessSG", {
            vpc: vpc,
            description: "Shared security group for all Lambdas that need database access",
            securityGroupName: "lambda-database-access-sg",
            allowAllOutbound: true,
        });
    }
}