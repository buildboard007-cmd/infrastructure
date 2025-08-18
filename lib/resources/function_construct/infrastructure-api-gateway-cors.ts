import {Construct} from "constructs";
import {FuncProps} from "../../types/func-props";
import {GoFunction} from "@aws-cdk/aws-lambda-go-alpha";
import * as path from 'path';
import {Duration} from "aws-cdk-lib";
import {GetRetentionDays} from "../../utils/lambda-utils";
import {ssmPolicy} from "../../utils/policy-utils";
import {GetAccountId} from "../../utils/account-utils";

export class InfrastructureApiGatewayCors extends Construct {

    private readonly _func: GoFunction;

    constructor(scope: Construct, id: string, props: FuncProps) {
        super(scope, id);

        const account = GetAccountId(props.stageEnvironment);
        const functionName = `${props?.options.githubRepo}-api-gateway-cors`

        this._func = new GoFunction(this, id, {
            entry: path.join(__dirname, `../../../src/infrastructure-api-gateway-cors`),
            functionName: functionName,
            timeout: Duration.seconds(10),
            environment: {
                LOG_LEVEL: "error",
                IS_LOCAL: "false",
            },
            logRetention: GetRetentionDays(props),
            bundling: {
                goBuildFlags: ['-ldflags "-s -w"'],
            },
        });

        this._func.addToRolePolicy(ssmPolicy(account));
    }

    get function(): GoFunction {
        return this._func
    }

    get functionArn(): string {
        return this._func.functionArn;
    }
}