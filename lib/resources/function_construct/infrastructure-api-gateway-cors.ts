import {Construct} from "constructs";
import {FuncProps} from "../../types/func-props";
import {GoFunction} from "@aws-cdk/aws-lambda-go-alpha";
import * as path from 'path';
import {Duration} from "aws-cdk-lib";
import {GetRetentionDays} from "../../utils/lambda-utils";
import {ssmPolicy} from "../../utils/policy-utils";
import {GetAccountId} from "../../utils/account-utils";
import {getBaseLambdaEnvironment} from "../../utils/lambda-environment";

export class InfrastructureApiGatewayCors extends Construct {

    private readonly func: GoFunction;

    constructor(scope: Construct, id: string, props: FuncProps) {
        super(scope, id);

        const account = GetAccountId(props.stageEnvironment);
        const functionName = `${props?.options.githubRepo}-api-gateway-cors`

        this.func = new GoFunction(this, id, {
            entry: path.join(__dirname, `../../../src/infrastructure-api-gateway-cors`),
            functionName: functionName,
            timeout: Duration.seconds(10),
            environment: getBaseLambdaEnvironment(props.stageEnvironment),
            logRetention: GetRetentionDays(props),
            bundling: {
                goBuildFlags: ['-ldflags "-s -w"'],
            },
        });

        this.func.addToRolePolicy(ssmPolicy());
    }

    get function(): GoFunction {
        return this.func
    }

    get functionArn(): string {
        return this.func.functionArn;
    }
}