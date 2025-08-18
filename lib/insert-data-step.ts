import {Effect, PolicyStatement} from "aws-cdk-lib/aws-iam";
import {CodeBuildStep} from "aws-cdk-lib/pipelines";
import {Construct} from "constructs";
import {StackOptions} from "./types/stack-options";

interface InsertDataProps {
    options: StackOptions;
    account: string;
}

export class InsertDataStep extends Construct {
    private readonly _codeBuildStep: CodeBuildStep;

    constructor(scope: Construct, id: string, props: InsertDataProps) {
        super(scope, id);
        this._codeBuildStep = new CodeBuildStep("InsertDataStep", {
            commands: [
                `export $(printf "AWS_ACCESS_KEY_ID=%s AWS_SECRET_ACCESS_KEY=%s AWS_SESSION_TOKEN=%s" $(aws sts assume-role --role-arn arn:aws:iam::${props.account}:role/${props.options.stackName}-DynamoInsertDataRole --role-session-name dynamo-insert-data --query "Credentials.[AccessKeyId,SecretAccessKey,SessionToken]" --output text))`,
                "npm install",
                "npm run ts-node scripts/dynamodb-insert-data.ts",
            ],
            rolePolicyStatements: [
                new PolicyStatement({
                    sid: `${props.options.stackName}DynamoInsertDataPolicy`,
                    effect: Effect.ALLOW,
                    actions: ["sts:AssumeRole"],
                    resources: [
                        `arn:aws:iam::${props.account}:role/${props.options.stackName}-DynamoInsertDataRole`,
                    ],
                }),
            ],
        });
    }

    get exportStep(): CodeBuildStep {
        return this._codeBuildStep;
    }
}
