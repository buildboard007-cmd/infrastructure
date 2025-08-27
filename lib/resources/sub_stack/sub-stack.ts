import {NestedStack, NestedStackProps} from "aws-cdk-lib";
import {findStageOption as findStageOptions, StackOptions} from "../../types/stack-options";
import {StageEnvironment} from "../../types/stage-environment";
import {Construct} from "constructs";
import {KeyConstruct} from "../key_construct/key-construct";
import {LambdaConstruct} from "../lambda_construct/lambda-construct";
import {LambdaConstructProps} from "../../types/lambda-construct-props";
import {CognitoConstruct} from "../cognito_construct/cognito-construct";
import * as path from "path";
import * as fs from "fs";
import {OpenApiBuilder} from "openapi3-ts";
import {ApiDefinition, BasePathMapping, DomainName, SpecRestApi} from "aws-cdk-lib/aws-apigateway";
import {ServicePrincipal} from "aws-cdk-lib/aws-iam";
import {GetAccountId} from "../../utils/account-utils";
import {addCors, addSecuritySchemeExtension} from "../../utils/api-utils";

interface KeyProps extends NestedStackProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
    builder: OpenApiBuilder;
}

export class SubStack extends NestedStack {
    private readonly keyConstruct: KeyConstruct;
    private readonly lambdaConstruct: LambdaConstruct;
    private readonly api: SpecRestApi;

    constructor(scope: Construct, id: string, props: KeyProps) {
        super(scope, id, props);

        // this.keyConstruct = new KeyConstruct(this, 'KeyConstruct', {
        //     stageEnvironment: props.stageEnvironment,
        // });

        const lambdaConstructProps: LambdaConstructProps = {
            options: props.options,
            stageEnvironment: props.stageEnvironment,
            builder: props.builder
        };

        this.lambdaConstruct = new LambdaConstruct(this, "LambdaConstruct", lambdaConstructProps);

        const cognitoConstruct = new CognitoConstruct(this, "CognitoConstruct", {
            options: props.options,
            tokenCustomizerLambda: this.lambdaConstruct.tokenCustomizerLambda,
            userSignupLambda: this.lambdaConstruct.userSignupLambda,
            stage: props.stageEnvironment,
        });

        const account = GetAccountId(props.stageEnvironment);
        
        // Add CORS support using infrastructure-api-gateway-cors Lambda
        const corsFunctionName = `${props.options.githubRepo}-api-gateway-cors`;
        const paths = Object.keys(props.builder.getSpec().paths);
        for (const path of paths) {
            addCors(path, props.builder, props.options.defaultRegion, account, corsFunctionName);
        }

        // Create API first without Cognito authentication to get base spec
        const file = path.join(__dirname, `../specs/spec.${account}.out.yaml`);
        fs.writeFileSync(file, props.builder.getSpecAsYaml());

        this.api = new SpecRestApi(this, "InfrastructureAPI", {
            apiDefinition: ApiDefinition.fromAsset(file),
            deployOptions: {
                stageName: props.options.apiStageName,
            },
            description: props.options.apiName,
            restApiName: props.options.apiName,
        });

        // Note: Cognito authentication will be handled at the Lambda level
        // to avoid CDK token resolution issues in the OpenAPI spec

        // Grant API Gateway permission to invoke the CORS Lambda
        this.lambdaConstruct.corsLambda.addPermission('ApiGatewayCorsPermission', {
            principal: new ServicePrincipal('apigateway.amazonaws.com'),
            sourceArn: this.api.arnForExecuteApi('*', '/*', '*'),
        });

        // Skip domain for LOCAL
        if (props.stageEnvironment != StageEnvironment.LOCAL) {
            const stageOptions = findStageOptions(
                props.options,
                props.stageEnvironment
            );

            // need to figure out how to fail a CDK synth
            if (!stageOptions) {
                console.log("Didn't find a domain for the stage");
                return;
            }

            let domainName = DomainName.fromDomainNameAttributes(
                this,
                "APIDomainName",
                {
                    domainName: stageOptions.domainName,
                    domainNameAliasTarget: stageOptions.domainNameAliasTarget,
                    domainNameAliasHostedZoneId: stageOptions.domainNameAliasHostedZoneId,
                }
            );

            const basePathMapping = new BasePathMapping(this, "ApiBasePathMapping", {
                domainName: domainName,
                restApi: this.api,
                basePath: "infra",
                stage: this.api.deploymentStage,
            });
        }

    }

    get corsLambdaArn(): string {
        return this.lambdaConstruct.corsLambdaArn;
    }

    // get dataKey(): IKey {
    //     return this.keyConstruct.dataKey;
    // }

    // get snsSqsKey(): IKey {
    //     return this.keyConstruct.snsSqsKey;
    // }
}
