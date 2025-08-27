import * as cdk from "aws-cdk-lib";
import {Construct} from "constructs";
import {StackOptions} from "./types/stack-options";
import {StageEnvironment} from "./types/stage-environment";
import {SubStack} from "./resources/sub_stack/sub-stack";
import * as fs from "fs";
import * as path from "path";
import * as yaml from "js-yaml";
import {GetAccountId} from "./utils/account-utils";
import {OpenApiBuilder, OpenAPIObject} from "openapi3-ts";
import {removeDiscriminators} from "./utils/api-utils";

interface LambdaStackProps extends cdk.StackProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
}

export class MainStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: LambdaStackProps) {
        super(scope, id, {
            ...props,
        });

        cdk.Tags.of(this).add("billingTag", props.options.serviceName);
        cdk.Tags.of(this).add("serviceName", props.options.serviceName);

        // const account = GetAccountId(props.stageEnvironment);
        const spec = yaml.load(
            fs.readFileSync(path.join(__dirname, "/resources/specs/source_spec.yaml"), "utf8")
        ) as OpenAPIObject;
        const builder = new OpenApiBuilder(spec);
        // Security scheme will be added in sub-stack after Cognito is created
        removeDiscriminators(builder);

        const subStack = new SubStack(this, `SubStack`, {
            options: props.options,
            stageEnvironment: props.stageEnvironment,
            builder: builder,
        });

        new cdk.CfnOutput(this, 'CorsLambdaArn', {
            value: subStack.corsLambdaArn,
            exportName: `api-gateway-cors-lambda-arn`,
            description: 'ARN of the CORS Lambda function for API Gateway'
        });
    }

}
