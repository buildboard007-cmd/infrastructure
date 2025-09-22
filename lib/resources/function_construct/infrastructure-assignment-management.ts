import { Construct } from 'constructs';
import { GoFunction } from '@aws-cdk/aws-lambda-go-alpha';
import { Duration } from 'aws-cdk-lib';
import { FuncProps } from '../../types/func-props';
import * as path from 'path';
import { GetRetentionDays } from '../../utils/lambda-utils';
import { getBaseLambdaEnvironment } from '../../utils/lambda-environment';
import { ssmPolicy } from '../../utils/policy-utils';

export class InfrastructureAssignmentManagement extends Construct {
    private readonly func: GoFunction;

    constructor(scope: Construct, id: string, props: FuncProps) {
        super(scope, id);

        const functionName = `${props?.options.githubRepo}-assignment-management`;

        this.func = new GoFunction(this, id, {
            entry: path.join(__dirname, `../../../src/infrastructure-assignment-management`),
            functionName: functionName,
            timeout: Duration.seconds(30),
            memorySize: 512,
            environment: getBaseLambdaEnvironment(props.stageEnvironment),
            logRetention: GetRetentionDays(props),
            bundling: {
                goBuildFlags: ['-ldflags "-s -w"'],
            },
        });

        this.func.addToRolePolicy(ssmPolicy());
    }

    get lambdaFunction(): GoFunction {
        return this.func;
    }

    get function(): GoFunction {
        return this.func;
    }

    get functionArn(): string {
        return this.func.functionArn;
    }
}