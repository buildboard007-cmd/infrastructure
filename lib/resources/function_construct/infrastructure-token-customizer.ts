import {Construct} from "constructs";
import {FuncProps} from "../../types/func-props";
import {GoFunction} from "@aws-cdk/aws-lambda-go-alpha";
import * as path from 'path';
import {Duration} from "aws-cdk-lib";
import {GetRetentionDays} from "../../utils/lambda-utils";
import {ssmPolicy} from "../../utils/policy-utils";
import {getBaseLambdaEnvironment} from "../../utils/lambda-environment";

export class InfrastructureTokenCustomizer extends Construct {

    private readonly func: GoFunction;

    constructor(scope: Construct, id: string, props: FuncProps) {
        super(scope, id);

        const functionName = `${props?.options.githubRepo}-token-customizer`

        // Build Lambda function configuration based on whether VPC config is provided
        const lambdaConfig: any = {
            entry: path.join(__dirname, `../../../src/infrastructure-token-customizer`),
            functionName: functionName,
            timeout: Duration.seconds(10),
            environment: getBaseLambdaEnvironment(props.stageEnvironment),
            logRetention: GetRetentionDays(props),
            bundling: {
                goBuildFlags: ['-ldflags "-s -w"'],
            },
        };

        // Add VPC configuration if provided (for database access)
        if (props.vpcConfig && props.sharedLambdaSG) {
            lambdaConfig.vpc = props.vpcConfig.vpc;
            lambdaConfig.vpcSubnets = props.vpcConfig.vpcSubnets;
            lambdaConfig.securityGroups = [props.sharedLambdaSG.securityGroup];
            lambdaConfig.allowPublicSubnet = props.vpcConfig.allowPublicSubnet;
        }

        this.func = new GoFunction(this, id, lambdaConfig);
        this.func.addToRolePolicy(ssmPolicy());
    }

    get function(): GoFunction {
        return this.func
    }

    get functionArn(): string {
        return this.func.functionArn;
    }
}