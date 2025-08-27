import {Construct} from "constructs";
import {FuncProps} from "../../types/func-props";
import {GoFunction} from "@aws-cdk/aws-lambda-go-alpha";
import * as path from 'path';
import {Duration} from "aws-cdk-lib";
import {addLambdaExtension} from "../../utils/api-utils";
import {GetRetentionDays} from "../../utils/lambda-utils";
import {GetAccountId} from "../../utils/account-utils";
import {getBaseLambdaEnvironment} from "../../utils/lambda-environment";

export class InfrastructureOrganizationManagement extends Construct {

    private readonly func: GoFunction;

    constructor(scope: Construct, id: string, props: FuncProps) {
        super(scope, id);

        const account = GetAccountId(props.stageEnvironment);
        const functionName = `${props?.options.githubRepo}-organization-management`

        this.func = new GoFunction(this, id, {
            entry: path.join(__dirname, `../../../src/infrastructure-organization-management`),
            functionName: functionName,
            timeout: Duration.seconds(10),
            environment: getBaseLambdaEnvironment(props.stageEnvironment),
            logRetention: GetRetentionDays(props),
            bundling: {
                goBuildFlags: ['-ldflags "-s -w"'],
            },
        })
        ;

        // Add PUT method for updating organization
        addLambdaExtension('/org',
            props.builder,
            props.options.defaultRegion,
            account,
            functionName,
            'put');
        
        // Add GET method for retrieving organization
        addLambdaExtension('/org',
            props.builder,
            props.options.defaultRegion,
            account,
            functionName,
            'get');

    }

    get function(): GoFunction {
        return this.func
    }

    get functionArn(): string {
        return this.func.functionArn;
    }
}