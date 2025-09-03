import {GoFunction} from "@aws-cdk/aws-lambda-go-alpha";
import {Construct} from "constructs";
import {FuncProps} from "../../types/func-props";
import {PolicyStatement} from "aws-cdk-lib/aws-iam";
import * as path from 'path';
import {Duration} from "aws-cdk-lib";
import {GetRetentionDays} from "../../utils/lambda-utils";
import {getBaseLambdaEnvironment} from "../../utils/lambda-environment";
import {ssmPolicy} from "../../utils/policy-utils";

export class InfrastructureUserManagement extends Construct {
    private readonly func: GoFunction;

    constructor(scope: Construct, id: string, props: FuncProps) {
        super(scope, id);

        const functionName = `${props?.options.githubRepo}-user-management`

        this.func = new GoFunction(this, id, {
            entry: path.join(__dirname, `../../../src/infrastructure-user-management`),
            functionName: functionName,
            timeout: Duration.seconds(10),
            environment: getBaseLambdaEnvironment(props.stageEnvironment),
            logRetention: GetRetentionDays(props),
            bundling: {
                goBuildFlags: ['-ldflags "-s -w"'],
            },
        });

        this.func.addToRolePolicy(ssmPolicy());

        // Add IAM permissions for Cognito User Pool management
        this.func.addToRolePolicy(new PolicyStatement({
            actions: [
                "cognito-idp:AdminCreateUser",
                "cognito-idp:AdminDeleteUser",
                "cognito-idp:AdminSetUserPassword",
                "cognito-idp:AdminInitiateAuth",
                "cognito-idp:AdminUpdateUserAttributes",
                "cognito-idp:AdminGetUser",
                "cognito-idp:ListUsers",
                "cognito-idp:AdminResetUserPassword"
            ],
            resources: ["*"]
        }));
    }

    get function(): GoFunction {
        return this.func
    }

    get functionArn(): string {
        return this.func.functionArn;
    }
}