/**
 * Centralized VPC configuration utility for database access
 * Following token-customizer pattern with hardcoded VPC configuration
 */

import * as ec2 from "aws-cdk-lib/aws-ec2";
import { Construct } from "constructs";

/**
 * Get database VPC configuration for Lambda functions
 * Returns the same hardcoded VPC configuration used across all Lambda functions
 */
export function getDatabaseVpcConfig(scope: Construct, id: string) {
    // Hardcoded VPC configuration following token-customizer pattern
    const vpc = ec2.Vpc.fromVpcAttributes(scope, `${id}-DatabaseVpc`, {
        vpcId: "vpc-0e6186e22705e39aa",
        availabilityZones: ["us-east-2a", "us-east-2b", "us-east-2c"],
        publicSubnetIds: [
            "subnet-0d997c4a8cf35f77c",
            "subnet-0f38126ed73ba4cc5",
            "subnet-0b95949ec310734eb"
        ],
    });

    // Import existing RDS security group
    const databaseSecurityGroup = ec2.SecurityGroup.fromSecurityGroupId(
        scope,
        `${id}-DatabaseSecurityGroup`,
        "sg-03e31c5306de8ebde" // RDS security group (postgres-sg)
    );

    // VPC subnet configuration for Lambda functions
    const vpcSubnets = {
        subnetType: ec2.SubnetType.PUBLIC,
    };

    return {
        vpc,
        databaseSecurityGroup,
        vpcSubnets,
        allowPublicSubnet: true,
    };
}