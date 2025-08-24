import * as iam from "aws-cdk-lib/aws-iam";
import {Effect, PolicyStatement} from "aws-cdk-lib/aws-iam";
import {Aws} from "aws-cdk-lib";

/**
 * SSM Parameter Store access policy
 * Updated to follow alerts-functions pattern with flexible region and environment support
 */
export const ssmPolicy = () => {
    return new iam.PolicyStatement({
        resources: [
            `arn:aws:ssm:us-west-2:${Aws.ACCOUNT_ID}:parameter/alerts-functions/*`,
            "*"
        ],
        effect: iam.Effect.ALLOW,
        actions: [
            "ssm:GetParametersByPath",
            "ssm:GetParameters",
            "ssm:GetParameter",
        ],
    });
}

/**
 * RDS access policy for Lambda functions that need database connectivity
 * Used when Lambda functions are deployed in same VPC as RDS
 */
export const rdsPolicy = (accountId: string): PolicyStatement => {
    return new iam.PolicyStatement({
        resources: [
            `arn:aws:rds:*:${accountId}:db:*`,
            `arn:aws:rds:*:${accountId}:cluster:*`,
        ],
        effect: Effect.ALLOW,
        actions: [
            "rds:DescribeDBInstances",
            "rds:DescribeDBClusters",
        ],
    });
};


