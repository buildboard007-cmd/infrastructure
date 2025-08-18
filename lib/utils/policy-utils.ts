import * as iam from "aws-cdk-lib/aws-iam";
import {Effect, PolicyStatement} from "aws-cdk-lib/aws-iam";

import {ITable} from "aws-cdk-lib/aws-dynamodb";
import {Aws} from "aws-cdk-lib";

export const ssmPolicy = (accountId: string) => {
    return new iam.PolicyStatement({
        resources: [
            `arn:aws:ssm:us-west-2:${accountId}:parameter/infrastructure/*`,
            "*",
        ],
        effect: iam.Effect.ALLOW,
        actions: [
            "ssm:GetParametersByPath",
            "ssm:GetParameters",
            "ssm:GetParameter",
        ],
    });
}


export const dynamodbPolicy = (table: ITable): PolicyStatement => {
    const tableArn = table.tableArn!;
    const tableIndexArn = `${tableArn}/index/*`
    const tableStreamArn = table.tableArn!;

    return new iam.PolicyStatement({
        resources: [
            tableArn,
            tableIndexArn,
            tableStreamArn,
        ],
        actions: [
            "dynamodb:BatchGetItem",
            "dynamodb:BatchWriteItem",
            "dynamodb:ConditionCheckItem",
            "dynamodb:DeleteItem",
            "dynamodb:DescribeTable",
            "dynamodb:GetItem",
            "dynamodb:GetRecords",
            "dynamodb:GetShardIterator",
            "dynamodb:PutItem",
            "dynamodb:Query",
            "dynamodb:Scan",
            "dynamodb:UpdateItem",
            "dynamodb:ListStreams",
            "dynamodb:DescribeStream",
            "dynamodb:GetRecords",
            "dynamodb:GetShardIterator"
        ],
        effect: Effect.ALLOW,
    });
};

export const cloudWatchPolicy = () => {
    return new iam.PolicyStatement({
        resources: ["*"],
        effect: Effect.ALLOW,
        actions: [
            "logs:CreateLogGroup",
            "logs:CreateLogStream",
            "logs:PutLogEvents",
            "logs:CreateLogDelivery",
            "logs:GetLogDelivery",
            "logs:UpdateLogDelivery",
            "logs:DeleteLogDelivery",
            "logs:ListLogDeliveries",
            "logs:PutResourcePolicy",
            "logs:DescribeResourcePolicies",
            "logs:DescribeLogGroups",
        ],
    });
};

export const stepFunctionPolicy = (): PolicyStatement => {
    return new iam.PolicyStatement({
        resources: ["*"],
        actions: [
            "states:DescribeExecution",
            "states:StartExecution",
            "states:StartSyncExecution",
            "states:StopExecution",
        ],
        effect: Effect.ALLOW,
    });
};

export const lambdaPolicy = (resources: string[]) => {
    return new iam.PolicyStatement({
        resources: resources,
        effect: Effect.ALLOW,
        actions: ["lambda:InvokeFunction"],
    });
};


