import * as cdk from "aws-cdk-lib";
import {Construct} from "constructs";
import {StackOptions} from "./types/stack-options";
import {StageEnvironment} from "./types/stage-environment";
import {MultiApiSubStack} from "./resources/sub_stack/multi-api-sub-stack";
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

        const multiApiSubStack = new MultiApiSubStack(this, `MultiApiSubStack`, {
            options: props.options,
            stageEnvironment: props.stageEnvironment,
        });

        new cdk.CfnOutput(this, 'CorsLambdaArn', {
            value: multiApiSubStack.corsLambdaArn,
            exportName: `api-gateway-cors-lambda-arn`,
            description: 'ARN of the CORS Lambda function for API Gateway'
        });

        // Output all API Gateway IDs and URLs for reference
        new cdk.CfnOutput(this, 'IamApiId', {
            value: multiApiSubStack.iamApiId,
            exportName: `iam-api-id`,
            description: 'API Gateway ID for IAM Management API'
        });

        new cdk.CfnOutput(this, 'ProjectsApiId', {
            value: multiApiSubStack.projectsApiId,
            exportName: `projects-api-id`,
            description: 'API Gateway ID for Projects Management API'
        });

        new cdk.CfnOutput(this, 'IssuesApiId', {
            value: multiApiSubStack.issuesApiId,
            exportName: `issues-api-id`,
            description: 'API Gateway ID for Issues Management API'
        });

        new cdk.CfnOutput(this, 'RfisApiId', {
            value: multiApiSubStack.rfisApiId,
            exportName: `rfis-api-id`,
            description: 'API Gateway ID for RFIs Management API'
        });

        // Output the API base URLs
        const stageName = props.options.apiStageName;
        const region = this.region;
        
        new cdk.CfnOutput(this, 'IamApiUrl', {
            value: `https://${multiApiSubStack.iamApiId}.execute-api.${region}.amazonaws.com/${stageName}`,
            exportName: `iam-api-url`,
            description: 'Base URL for IAM Management API'
        });

        new cdk.CfnOutput(this, 'ProjectsApiUrl', {
            value: `https://${multiApiSubStack.projectsApiId}.execute-api.${region}.amazonaws.com/${stageName}`,
            exportName: `projects-api-url`,
            description: 'Base URL for Projects Management API'
        });

        new cdk.CfnOutput(this, 'IssuesApiUrl', {
            value: `https://${multiApiSubStack.issuesApiId}.execute-api.${region}.amazonaws.com/${stageName}`,
            exportName: `issues-api-url`,
            description: 'Base URL for Issues Management API'
        });

        new cdk.CfnOutput(this, 'RfisApiUrl', {
            value: `https://${multiApiSubStack.rfisApiId}.execute-api.${region}.amazonaws.com/${stageName}`,
            exportName: `rfis-api-url`,
            description: 'Base URL for RFIs Management API'
        });
    }

}
